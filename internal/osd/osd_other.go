//go:build !darwin

// Package osd shows the system's own brightness indicator on the screen.
package osd

// Available reports whether this system has an indicator monctl can show.
func Available() bool { return false }

// Brightness does nothing on this system.
func Brightness(int, int) {}
