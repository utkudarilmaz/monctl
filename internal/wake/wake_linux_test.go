package wake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasSessionBus(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "bus")

	if hasSessionBus("", socket) || hasSessionBus("autolaunch:", socket) {
		t.Fatal("found a session bus where there is none")
	}
	if !hasSessionBus("unix:path=/run/user/1000/bus", socket) {
		t.Fatal("the address from the environment was not used")
	}
	if err := os.WriteFile(socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !hasSessionBus("", socket) {
		t.Fatal("the bus socket of the user was not found")
	}
}
