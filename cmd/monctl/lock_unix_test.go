//go:build unix

package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLockMonitorMakesCallersTakeTurns(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	unlock := lockMonitor()
	got := make(chan struct{})
	go func() {
		release := lockMonitor()
		close(got)
		release()
	}()

	select {
	case <-got:
		t.Fatal("a second caller got the lock while the first held it")
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	select {
	case <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("the second caller never got the lock")
	}
}

func TestLockPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if got := filepath.Dir(lockPath()); got != "/run/user/1000" {
		t.Fatalf("lock dir = %q", got)
	}
}

func TestLockMonitorWithoutAUsableFileStillReturns(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(t.TempDir(), "missing"))
	lockMonitor()()
}
