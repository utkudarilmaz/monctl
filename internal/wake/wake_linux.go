// Package wake turns on the screen of this computer.
package wake

import (
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

const settle = 2 * time.Second

// Screen wakes the screen of the GNOME session of this user and waits a
// moment for the picture to come up.
func Screen() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return errors.New("wake: no desktop session found for this user")
	}
	display := conn.Object("org.gnome.Mutter.DisplayConfig", "/org/gnome/Mutter/DisplayConfig")
	if err := display.Call("org.freedesktop.DBus.Properties.Set", 0,
		"org.gnome.Mutter.DisplayConfig", "PowerSaveMode", dbus.MakeVariant(int32(0))).Err; err != nil {
		return fmt.Errorf("wake: cannot turn the screen on, is this user logged in at the screen: %w", err)
	}
	saver := conn.Object("org.gnome.ScreenSaver", "/org/gnome/ScreenSaver")
	if err := saver.Call("org.gnome.ScreenSaver.SetActive", 0, false).Err; err != nil {
		return fmt.Errorf("wake: cannot leave the blank screen: %w", err)
	}
	time.Sleep(settle)
	return nil
}
