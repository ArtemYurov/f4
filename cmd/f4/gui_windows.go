//go:build windows

package main

import (
	"strings"

	"github.com/unxed/f4/internal/config"
	"github.com/unxed/f4/internal/plughost"
	"github.com/unxed/vtui"
)

func RunGui(backend string) error {
	return withGUIRuntime(func() error {
		if backend == "qt" || strings.HasPrefix(backend, "ext:") {
			return plughost.RunExternalUIWithMapping(backend)
		}
		if err := checkGUIBackendAvailability(backend); err != nil {
			return err
		}
		stopIconManager := startWindowsWindowIconManager()
		defer stopIconManager()
		return vtui.RunInGUIWindow(config.App.GuiCols, config.App.GuiRows, backend, effectiveGuiFont(), float64(config.App.GuiFontSize), func() {
			SetupUI()
			openDashEFileIfRequested()
			restoreGuiWindowPosition()
		})
	})
}
