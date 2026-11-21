//go:build !unix

// Package remote lets a short monctl command hand a brightness change to the
// monctl tray that is already running.
package remote

import "errors"

// ErrNoServer means no tray is listening.
var ErrNoServer = errors.New("remote: no running tray")

// Handler applies a command and returns the new brightness and the maximum.
type Handler func(command string, value int) (current, limit int, err error)

// Path returns no path on this system.
func Path() string { return "" }

// Serve is not supported on this system.
func Serve(string, Handler) (func(), error) { return func() {}, nil }

// Send always reports that no tray is running.
func Send(string, string, int) (int, int, error) { return 0, 0, ErrNoServer }
