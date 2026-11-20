//go:build !linux

package display

import (
	"errors"

	"github.com/utkudarilmaz/monctl/internal/bus"
)

var errUnsupported = errors.New("display: this system is not supported yet")

// Detect returns the external displays attached to this machine.
func Detect() ([]Display, error) {
	return nil, errUnsupported
}

// Open opens the control channel of a display found by Detect.
func Open(Display) (bus.Bus, error) {
	return nil, errUnsupported
}
