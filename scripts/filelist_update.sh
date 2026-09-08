#!/bin/bash

# Every path below is relative to the repository root, so go there rather
# than requiring the caller to be standing in it.
cd "$(dirname "$0")/.." || exit 1

OUTPUT="docs/FILELIST.md"

echo "# Project Structure" > "$OUTPUT"
echo "" >> "$OUTPUT"

if command -v tree &> /dev/null; then
    tree -a -I ".git|$OUTPUT" | sed 's/^/    /' >> "$OUTPUT"
else
    find . -path './.git' -prune -o -name "$OUTPUT" -prune -o -print | sed 's/^/    /' >> "$OUTPUT"
fi

echo "File list updated in $OUTPUT"
