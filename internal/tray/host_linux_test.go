package tray

import (
	"strings"
	"testing"
)

func TestCheckHostWithoutASessionBus(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+t.TempDir()+"/none")

	err := checkHost()
	if err == nil || !strings.Contains(err.Error(), "no desktop session found") {
		t.Fatalf("checkHost = %v, want no desktop session", err)
	}
}
