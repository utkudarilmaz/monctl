// Package wake turns on the screen of this computer.
package wake

import (
	"fmt"
	"os/exec"
)

// Screen wakes the screen by declaring user activity and waits a moment for
// the picture to come up.
func Screen() error {
	if out, err := exec.Command("/usr/bin/caffeinate", "-u", "-t", "2").CombinedOutput(); err != nil {
		return fmt.Errorf("wake: caffeinate failed: %w: %s", err, out)
	}
	return nil
}
