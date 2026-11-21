//go:build unix

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/utkudarilmaz/monctl/internal/remote"
)

// lockMonitor waits until no other monctl of this user talks to a monitor.
// Two programs on the control channel at once read each other's replies.
// Without a usable lock file it lets the caller go on.
func lockMonitor() func() {
	f, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return func() {}
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return func() {}
	}
	return func() { _ = f.Close() }
}

func lockPath() string {
	dir := remote.Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, fmt.Sprintf("monctl-%d.lock", os.Getuid()))
}
