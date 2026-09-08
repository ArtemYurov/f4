package archive

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
)

const sfxProbeLimit = 64 << 20

type sfxSignature struct {
	magic  []byte
	format string
	suffix string
}

var sfxSignatures = []sfxSignature{
	{magic: []byte("PK\x03\x04"), format: "zip", suffix: ".zip"},
	{magic: []byte("PK\x05\x06"), format: "zip", suffix: ".zip"},
	{magic: []byte("7z\xBC\xAF\x27\x1C"), format: "fallback", suffix: ".7z"},
	{magic: []byte("Rar!\x1A\x07\x00"), format: "fallback", suffix: ".rar"},
	{magic: []byte("Rar!\x1A\x07\x01\x00"), format: "fallback", suffix: ".rar"},
}

type embeddedArchive struct {
	format string
	suffix string
	offset int64
}

func findEmbeddedArchive(filename string) (embeddedArchive, bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return embeddedArchive{}, false, err
	}
	defer func() { _ = file.Close() }()

	const chunkSize = 64 << 10
	maxMagic := 0
	for _, signature := range sfxSignatures {
		if len(signature.magic) > maxMagic {
			maxMagic = len(signature.magic)
		}
	}

	chunk := make([]byte, chunkSize)
	var carry []byte
	var scanned int64
	for scanned < sfxProbeLimit {
		want := int64(len(chunk))
		if remaining := sfxProbeLimit - scanned; remaining < want {
			want = remaining
		}
		n, readErr := file.Read(chunk[:int(want)])
		if n > 0 {
			block := make([]byte, 0, len(carry)+n)
			block = append(block, carry...)
			block = append(block, chunk[:n]...)
			blockStart := scanned - int64(len(carry))

			bestIndex := -1
			var best sfxSignature
			for _, signature := range sfxSignatures {
				index := bytes.Index(block, signature.magic)
				if index >= 0 && (bestIndex < 0 || index < bestIndex) {
					bestIndex = index
					best = signature
				}
			}
			if bestIndex >= 0 {
				return embeddedArchive{
					format: best.format,
					suffix: best.suffix,
					offset: blockStart + int64(bestIndex),
				}, true, nil
			}

			keep := maxMagic - 1
			if len(block) > keep {
				carry = append(carry[:0], block[len(block)-keep:]...)
			} else {
				carry = append(carry[:0], block...)
			}
			scanned += int64(n)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return embeddedArchive{}, false, readErr
		}
	}

	return embeddedArchive{}, false, nil
}

type sfxBacking struct {
	path string
	once sync.Once
	err  error
}

func (b *sfxBacking) Close() error {
	b.once.Do(func() {
		b.err = os.Remove(b.path)
		if errors.Is(b.err, os.ErrNotExist) {
			b.err = nil
		}
	})
	return b.err
}

func materializeEmbeddedArchive(filename string, embedded embeddedArchive) (string, io.Closer, error) {
	if embedded.offset <= 0 {
		return filename, nil, nil
	}

	source, err := os.Open(filename)
	if err != nil {
		return "", nil, err
	}
	stat, err := source.Stat()
	if err != nil {
		_ = source.Close()
		return "", nil, err
	}
	if embedded.offset >= stat.Size() {
		_ = source.Close()
		return "", nil, os.ErrInvalid
	}

	target, err := os.CreateTemp("", "f4-sfx-*"+embedded.suffix)
	if err != nil {
		_ = source.Close()
		return "", nil, err
	}
	targetName := target.Name()
	cleanup := func() {
		_ = source.Close()
		_ = target.Close()
		_ = os.Remove(targetName)
	}

	_, copyErr := io.Copy(target, io.NewSectionReader(source, embedded.offset, stat.Size()-embedded.offset))
	closeTargetErr := target.Close()
	closeSourceErr := source.Close()
	if copyErr != nil || closeTargetErr != nil || closeSourceErr != nil {
		cleanup()
		if copyErr != nil {
			return "", nil, copyErr
		}
		if closeTargetErr != nil {
			return "", nil, closeTargetErr
		}
		return "", nil, closeSourceErr
	}

	return targetName, &sfxBacking{path: targetName}, nil
}
