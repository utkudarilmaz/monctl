// Package keys catches the brightness keys of the keyboard.
package keys

/*
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices
#include "keys_darwin.h"
*/
import "C"

import "errors"

// ErrNotTrusted means macOS has not allowed this program to see key presses.
var ErrNotTrusted = errors.New(`keys: macOS does not let monctl see the brightness keys yet.

Open System Settings, Privacy and Security, Accessibility, and turn monctl on.
Then start the tray again`)

var (
	onStep   func(direction int)
	isUsable func() bool
	onMissed func()
)

// Start catches the brightness keys. While usable reports true, a key press
// calls step with 1 or -1 and the key does nothing else. Otherwise the key
// keeps its normal job and missed is called. The three functions run on the
// main thread and must return at once.
func Start(step func(direction int), usable func() bool, missed func()) error {
	onStep, isUsable, onMissed = step, usable, missed
	switch C.monctlStartTap() {
	case 0:
		return nil
	case 1:
		return ErrNotTrusted
	default:
		return errors.New("keys: cannot watch the keyboard")
	}
}

//export monctlBrightnessKey
func monctlBrightnessKey(direction, down C.int) C.int {
	if !isUsable() {
		if down != 0 {
			onMissed()
		}
		return 0
	}
	if down != 0 {
		onStep(int(direction))
	}
	return 1
}
