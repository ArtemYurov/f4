//go:build !windows

package main

// The X11 window over the terminal, for a terminal that cannot show a picture
// itself. See docs on the image viewer's last resort.
func (termApplication) InstallImageOverlay() { InstallX11Overlay() }
