package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/unxed/vtui"
)

// These flags mirror the useful part of Far's Change Drive Menu Options.
// They are intentionally kept as one persisted bit field so old profiles can
// carry the setting without another configuration section.
const (
	driveMenuShowType uint32 = 1 << iota
	driveMenuShowLabel
	driveMenuUseShellName
	driveMenuShowFilesystem
	driveMenuShowSize
	driveMenuShowSizeFloat
	driveMenuShowNetworkName
	driveMenuShowPlugins
	driveMenuSortPluginsByHotkey
	driveMenuShowRemovable
	driveMenuShowCD
	driveMenuShowRemote
	driveMenuDetectVirtual
	driveMenuShowBookmarks
)

const defaultDriveMenuOptions = driveMenuShowType |
	driveMenuShowLabel |
	driveMenuShowFilesystem |
	driveMenuShowSize |
	driveMenuShowSizeFloat |
	driveMenuShowPlugins |
	driveMenuShowRemovable |
	driveMenuShowCD |
	driveMenuShowRemote |
	driveMenuDetectVirtual |
	driveMenuShowBookmarks

func parseDriveMenuOptions(value string) uint32 {
	if strings.TrimSpace(value) == "" {
		return defaultDriveMenuOptions
	}
	var options uint32
	if _, err := fmt.Sscanf(value, "%d", &options); err != nil {
		return defaultDriveMenuOptions
	}
	return options
}

type driveMenuKind uint8

const (
	driveMenuKindUnknown driveMenuKind = iota
	driveMenuKindFixed
	driveMenuKindRemovable
	driveMenuKindCD
	driveMenuKindRemote
	driveMenuKindPhysical
)

type driveMenuOptionSpec struct {
	flag  uint32
	label string
}

var driveMenuOptionSpecs = []driveMenuOptionSpec{
	{driveMenuShowType, "Drive.ShowType"},
	{driveMenuShowLabel, "Drive.ShowLabel"},
	{driveMenuUseShellName, "Drive.UseShellName"},
	{driveMenuShowFilesystem, "Drive.ShowFilesystem"},
	{driveMenuShowSize, "Drive.ShowSize"},
	{driveMenuShowSizeFloat, "Drive.ShowSizeFloat"},
	{driveMenuShowNetworkName, "Drive.ShowNetworkName"},
	{driveMenuShowPlugins, "Drive.ShowPlugins"},
	{driveMenuSortPluginsByHotkey, "Drive.SortPluginsByHotkey"},
	{driveMenuShowRemovable, "Drive.ShowRemovable"},
	{driveMenuShowCD, "Drive.ShowCD"},
	{driveMenuShowRemote, "Drive.ShowRemote"},
	{driveMenuDetectVirtual, "Drive.DetectVirtual"},
	{driveMenuShowBookmarks, "Drive.ShowBookmarks"},
}

func driveMenuOptionEnabled(options, flag uint32) bool { return options&flag != 0 }

func driveMenuNameWithoutMarker(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, "&", ""))
}

func driveMenuBaseName(name string) string {
	name = driveMenuNameWithoutMarker(name)
	switch {
	case strings.HasPrefix(name, "/ Root"):
		return "/"
	case strings.HasPrefix(name, "~ Home"):
		return "~"
	case len(name) >= 2 && name[1] == ':':
		return name[:2]
	default:
		return name
	}
}

func driveMenuInfoPath(name string) string {
	clean := driveMenuNameWithoutMarker(name)
	switch {
	case strings.HasPrefix(clean, "/ Root"):
		return "/"
	case strings.HasPrefix(clean, "~ Home"):
		home, _ := os.UserHomeDir()
		return home
	case len(clean) >= 2 && clean[1] == ':':
		// GetDiskFreeSpaceEx and GetVolumeInformation both want a root.
		return clean[:2] + string(os.PathSeparator)
	default:
		return ""
	}
}

func driveMenuKindFor(name, path string) driveMenuKind {
	if strings.Contains(strings.ToLower(name), "physical disk") {
		return driveMenuKindPhysical
	}
	if path == "" {
		return driveMenuKindUnknown
	}
	return driveMenuPlatformKind(path)
}

func driveMenuKindLabel(kind driveMenuKind) string {
	switch kind {
	case driveMenuKindFixed:
		return Msg("Drive.TypeFixed")
	case driveMenuKindRemovable:
		return Msg("Drive.TypeRemovable")
	case driveMenuKindCD:
		return Msg("Drive.TypeCD")
	case driveMenuKindRemote:
		return Msg("Drive.TypeRemote")
	case driveMenuKindPhysical:
		return Msg("Drive.TypePhysical")
	default:
		return ""
	}
}

func driveMenuSize(b uint64, decimal bool) string {
	if decimal {
		return formatBytesHuman(b)
	}
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(1024), 0
	for n := b / 1024; n >= 1024 && exp < 6; n /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%d %ciB", b/div, "KMGTPE"[exp])
}

// driveMenuPlatformItemText renders the platform rows as Far-style columns.
// fsInfo is deliberately used only for built-in local rows: plugin VFSes can
// be remote and may block while resolving their metadata.
func driveMenuPlatformItemText(drv DriveEntry, options uint32) string {
	base := driveMenuBaseName(drv.Name)
	path := driveMenuInfoPath(drv.Name)
	kind := driveMenuKindFor(drv.Name, path)
	parts := []string{base}

	if driveMenuOptionEnabled(options, driveMenuShowType) {
		if label := driveMenuKindLabel(kind); label != "" {
			parts = append(parts, label)
		}
	}

	info, infoOK := FSInfo{}, false
	if path != "" {
		info, infoOK = fsInfo(path)
	}
	if infoOK {
		if driveMenuOptionEnabled(options, driveMenuShowLabel) && info.Label != "" {
			parts = append(parts, info.Label)
		}
		if driveMenuOptionEnabled(options, driveMenuShowFilesystem) && info.Type != "" {
			parts = append(parts, info.Type)
		}
		if driveMenuOptionEnabled(options, driveMenuShowSize) {
			decimal := driveMenuOptionEnabled(options, driveMenuShowSizeFloat)
			parts = append(parts, driveMenuSize(info.Total, decimal), driveMenuSize(info.Free, decimal))
		}
		if driveMenuOptionEnabled(options, driveMenuShowNetworkName) && info.Mount != "" && info.Mount != path {
			parts = append(parts, info.Mount)
		}
	}
	return strings.Join(parts, " | ")
}

func driveMenuPlatformItemVisible(drv DriveEntry, options uint32) bool {
	kind := driveMenuKindFor(drv.Name, driveMenuInfoPath(drv.Name))
	switch kind {
	case driveMenuKindRemovable:
		return driveMenuOptionEnabled(options, driveMenuShowRemovable)
	case driveMenuKindCD:
		return driveMenuOptionEnabled(options, driveMenuShowCD)
	case driveMenuKindRemote:
		return driveMenuOptionEnabled(options, driveMenuShowRemote)
	default:
		return true
	}
}

func (pf *PanelsFrame) openDriveMenuOptions(panelIdx int, menu *vtui.VMenu) {
	const width, height = 78, 21
	dlg := vtui.NewCenteredDialog(width, height, Msg("Drive.OptionsTitle"))
	dlg.ShowClose = true

	options := AppConfig.DriveMenuOptions
	checks := make([]*vtui.Checkbox, 0, len(driveMenuOptionSpecs))
	for _, spec := range driveMenuOptionSpecs {
		check := vtui.NewCheckbox(0, 0, Msg(spec.label), false)
		if driveMenuOptionEnabled(options, spec.flag) {
			check.State = 1
		}
		checks = append(checks, check)
		dlg.AddItem(check)
	}

	ok := vtui.NewButton(0, 0, Msg("vtui.Ok"))
	ok.IsDefault = true
	cancel := vtui.NewButton(0, 0, Msg("vtui.Cancel"))
	dlg.AddItem(ok)
	dlg.AddItem(cancel)

	vbox := vtui.NewVBoxLayout(dlg.X1+2, dlg.Y1+2, width-4, height-4)
	for _, check := range checks {
		vbox.Add(check, vtui.Margins{}, vtui.AlignLeft)
	}
	buttons := vtui.NewHBoxLayout(0, 0, width-4, 1)
	buttons.HorizontalAlign = vtui.AlignCenter
	buttons.Spacing = 2
	buttons.Add(ok, vtui.Margins{}, vtui.AlignTop)
	buttons.Add(cancel, vtui.Margins{}, vtui.AlignTop)
	vbox.Add(buttons, vtui.Margins{Top: 1}, vtui.AlignFill)
	vbox.Apply()
	dlg.SetFocusedItem(checks[0])

	cancel.OnClick = func() { dlg.Close() }
	ok.OnClick = func() {
		var updated uint32
		for i, check := range checks {
			if check.State == 1 {
				updated |= driveMenuOptionSpecs[i].flag
			}
		}
		AppConfig.DriveMenuOptions = updated
		SaveConfig()
		pos := menu.SelectPos
		dlg.Close()
		menu.Close()
		vtui.FrameManager.PostTask(func() { pf.showDriveMenuAt(panelIdx, pos) })
	}

	vtui.FrameManager.Push(dlg)
}
