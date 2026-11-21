//go:build !darwin

// Package keys catches the brightness keys of the keyboard.
package keys

// Start does nothing on this system. The desktop's own shortcuts run monctl.
func Start(func(direction int), func() bool, func()) error { return nil }
