//go:build windows

package main

import "golang.org/x/sys/windows"

func driveMenuPlatformKind(path string) driveMenuKind {
	root, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return driveMenuKindUnknown
	}
	switch windows.GetDriveType(root) {
	case windows.DRIVE_FIXED, windows.DRIVE_RAMDISK:
		return driveMenuKindFixed
	case windows.DRIVE_REMOVABLE:
		return driveMenuKindRemovable
	case windows.DRIVE_CDROM:
		return driveMenuKindCD
	case windows.DRIVE_REMOTE:
		return driveMenuKindRemote
	default:
		return driveMenuKindUnknown
	}
}
