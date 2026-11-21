package keys

import "testing"

func TestOnGnome(t *testing.T) {
	for desktop, want := range map[string]bool{"GNOME": true, "ubuntu:GNOME": true, "gnome": true, "KDE": false, "": false, "GNOME-Flashback": false} {
		if got := onGnome(desktop); got != want {
			t.Errorf("onGnome(%q) = %v, want %v", desktop, got, want)
		}
	}
}
