// Package osd shows the system's own brightness indicator on the screen.
package osd

import (
	"os"

	"github.com/godbus/dbus/v5"
)

const (
	shellName = "org.gnome.Shell"
	shellPath = "/io/github/utkudarilmaz/monctl/Indicator"
	showCall  = "io.github.utkudarilmaz.monctl.Indicator.ShowBrightness"
)

// Available reports whether this system has an indicator monctl can show.
func Available() bool { return os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" }

// Brightness asks the monctl GNOME Shell extension to show the indicator for
// a brightness of value out of limit. Without the extension nothing happens.
func Brightness(value, limit int) {
	if !Available() {
		return
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		return
	}
	conn.Object(shellName, shellPath).Call(showCall, dbus.FlagNoReplyExpected|dbus.FlagNoAutoStart, Fraction(value, limit))
}
