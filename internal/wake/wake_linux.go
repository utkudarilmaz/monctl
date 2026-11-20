// Package wake turns on the screen of this computer.
package wake

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	settle      = 2 * time.Second
	callTimeout = 5 * time.Second
)

// Screen wakes the screen of the GNOME session of this user and waits a
// moment for the picture to come up.
func Screen() error {
	noSession := errors.New("wake: no desktop session found for this user")
	if !hasSessionBus(os.Getenv("DBUS_SESSION_BUS_ADDRESS"), fmt.Sprintf("/run/user/%d/bus", os.Getuid())) {
		return noSession
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		return noSession
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()

	display := conn.Object("org.gnome.Mutter.DisplayConfig", "/org/gnome/Mutter/DisplayConfig")
	if err := display.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Set", 0,
		"org.gnome.Mutter.DisplayConfig", "PowerSaveMode", dbus.MakeVariant(int32(0))).Err; err != nil {
		return fmt.Errorf("wake: cannot turn the screen on, is this user logged in at the screen: %w", err)
	}
	saver := conn.Object("org.gnome.ScreenSaver", "/org/gnome/ScreenSaver")
	if err := saver.CallWithContext(ctx, "org.gnome.ScreenSaver.SetActive", 0, false).Err; err != nil {
		return fmt.Errorf("wake: cannot leave the blank screen: %w", err)
	}
	time.Sleep(settle)
	return nil
}

// hasSessionBus reports whether a session bus is there to connect to. Without
// one the dbus library would start a new bus that nothing ever stops.
func hasSessionBus(address, socket string) bool {
	if address != "" && address != "autolaunch:" {
		return true
	}
	_, err := os.Stat(socket)
	return err == nil
}
