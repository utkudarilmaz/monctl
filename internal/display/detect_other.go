//go:build !linux

package display

import "errors"

// Detect returns the external displays attached to this machine.
func Detect() ([]Display, error) {
	return nil, errors.New("display: this system is not supported yet")
}
