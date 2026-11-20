//go:build !linux && !darwin

// Package wake turns on the screen of this computer.
package wake

import "errors"

// Screen is not supported on this system.
func Screen() error {
	return errors.New("wake: this system is not supported")
}
