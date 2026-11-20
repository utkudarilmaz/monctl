package tray

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const watcherName = "org.kde.StatusNotifierWatcher"

// ErrNoTrayHost means the desktop has nothing that shows tray icons.
var ErrNoTrayHost = errors.New(`tray: this desktop does not show tray icons.

On GNOME, install and turn on the AppIndicator extension:

  sudo apt install gnome-shell-extension-appindicator
  gnome-extensions enable ubuntu-appindicators@ubuntu.com

Log out and in again, then run "monctl tray" once more`)

func checkHost() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("tray: no desktop session found, run \"monctl tray\" from your desktop: %w", err)
	}
	var owned bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, watcherName).Store(&owned); err != nil {
		return fmt.Errorf("tray: cannot ask the session bus for a tray host: %w", err)
	}
	if !owned {
		return ErrNoTrayHost
	}
	return nil
}
