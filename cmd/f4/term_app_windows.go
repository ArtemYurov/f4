//go:build windows

package main

// The window over the console, for a console that cannot show a picture
// itself — which is conhost, where cmd.exe lives. Windows Terminal renders
// sixel and is left alone.
func (termApplication) InstallImageOverlay() { InstallConsoleOverlay() }
