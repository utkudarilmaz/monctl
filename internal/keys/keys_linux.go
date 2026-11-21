// Package keys catches the brightness keys of the keyboard.
package keys

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/utkudarilmaz/monctl/internal/names"
)

const (
	markerName      = "shortcuts"
	settingsTimeout = 5 * time.Second
)

// Start binds the brightness keys to monctl in the GNOME settings, once. A
// normal program cannot catch keys on GNOME, so the desktop runs monctl.
func Start(func(direction int), func() bool, func()) error {
	state := names.Path()
	program, err := os.Executable()
	if state == "" || err != nil {
		return nil
	}
	if _, err := exec.LookPath("gsettings"); err != nil {
		return nil
	}
	backlights, _ := filepath.Glob("/sys/class/backlight/*")
	return register(gsettings, program, filepath.Join(filepath.Dir(state), markerName), len(backlights) > 0)
}

func gsettings(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), settingsTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gsettings", args...).Output()
	return strings.TrimSpace(string(out)), err
}
