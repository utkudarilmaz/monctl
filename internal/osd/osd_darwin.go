// Package osd shows the system's own brightness indicator on the screen.
package osd

/*
#cgo LDFLAGS: -framework Cocoa
#include "osd_darwin.h"
*/
import "C"

// Available reports whether this system has an indicator monctl can show.
func Available() bool { return C.monctlOSDAvailable() != 0 }

// Brightness shows the indicator for a brightness of value out of limit.
func Brightness(value, limit int) {
	filled, total := Chiclets(value, limit)
	C.monctlOSDShow(C.int(filled), C.int(total))
}
