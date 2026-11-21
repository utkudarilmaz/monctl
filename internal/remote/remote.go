//go:build unix

// Package remote lets a short monctl command hand a brightness change to the
// monctl tray that is already running, over a socket only this user can open.
package remote

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	timeout = 3 * time.Second
	maxLine = 64
)

// ErrNoServer means no tray is listening.
var ErrNoServer = errors.New("remote: no running tray")

// Handler applies a command, "step" or "set", and returns the new brightness
// and the monitor's maximum.
type Handler func(command string, value int) (current, limit int, err error)

// Dir returns a directory that only this user can use and that is the same
// for every monctl of this user, however it was started. It is created when
// missing, and empty when there is no such directory.
func Dir() string {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(dir) {
		return dir
	}
	name := "monctl"
	base, err := os.UserCacheDir()
	if err != nil {
		base, name = os.TempDir(), fmt.Sprintf("monctl-%d", os.Getuid())
	}
	dir := filepath.Join(base, name)
	_ = os.MkdirAll(dir, 0o700)
	if !private(dir) {
		return ""
	}
	return dir
}

// private reports whether dir is a real directory of this user that nobody
// else can open. Another user can make the directory first in a shared place.
func private(dir string) bool {
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() {
		return false
	}
	return info.Mode().Perm()&0o077 == 0 || os.Chmod(dir, 0o700) == nil
}

// Path returns the socket path of this user, or nothing without a directory.
func Path() string {
	dir := Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, fmt.Sprintf("monctl-%d.sock", os.Getuid()))
}

// Serve listens on path until the returned function is called.
func Serve(path string, handle Handler) (func(), error) {
	if path == "" {
		return nil, errors.New("remote: no private directory for the socket")
	}
	if conn, err := net.DialTimeout("unix", path, timeout); err == nil {
		_ = conn.Close()
		return nil, errors.New("remote: another tray is already running")
	}
	_ = os.Remove(path)

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go answer(conn, handle)
		}
	}()
	return func() { _ = listener.Close() }, nil
}

func answer(conn net.Conn, handle Handler) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	line, err := bufio.NewReader(io.LimitReader(conn, maxLine)).ReadString('\n')
	if err != nil {
		return
	}
	var command string
	var value int
	if _, err := fmt.Sscanf(line, "%s %d", &command, &value); err != nil || (command != "step" && command != "set") {
		_, _ = fmt.Fprintln(conn, "err bad request")
		return
	}
	current, limit, err := handle(command, value)
	if err != nil {
		_, _ = fmt.Fprintf(conn, "err %s\n", strings.ReplaceAll(err.Error(), "\n", " "))
		return
	}
	_, _ = fmt.Fprintf(conn, "ok %d %d\n", current, limit)
}

// Send asks the tray at path to apply a command.
func Send(path, command string, value int) (current, limit int, err error) {
	if path == "" {
		return 0, 0, ErrNoServer
	}
	conn, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return 0, 0, ErrNoServer
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := fmt.Fprintf(conn, "%s %d\n", command, value); err != nil {
		return 0, 0, err
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return 0, 0, err
	}
	if text, failed := strings.CutPrefix(strings.TrimSpace(line), "err "); failed {
		return 0, 0, errors.New(text)
	}
	if _, err := fmt.Sscanf(line, "ok %d %d", &current, &limit); err != nil {
		return 0, 0, fmt.Errorf("remote: unexpected answer %q", strings.TrimSpace(line))
	}
	return current, limit, nil
}
