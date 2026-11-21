//go:build unix

package remote

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func socket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "monctl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

func TestSendReachesTheHandler(t *testing.T) {
	path := socket(t)
	var got []string
	stop, err := Serve(path, func(command string, value int) (int, int, error) {
		got = append(got, command)
		if value == 999 {
			return 0, 0, errors.New("monitor is\nsilent")
		}
		return 40 + value, 100, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	if current, limit, err := Send(path, "step", -5); err != nil || current != 35 || limit != 100 {
		t.Fatalf("Send step = %d, %d, %v", current, limit, err)
	}
	if current, _, err := Send(path, "set", 20); err != nil || current != 60 {
		t.Fatalf("Send set = %d, %v", current, err)
	}
	if _, _, err := Send(path, "set", 999); err == nil || err.Error() != "monitor is silent" {
		t.Fatalf("handler error = %v", err)
	}
	if _, _, err := Send(path, "reboot", 1); err == nil || len(got) != 3 {
		t.Fatalf("an unknown command must fail and not reach the handler: %v, calls %v", err, got)
	}

	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode = %v, %v; want 0600", info.Mode().Perm(), err)
	}
}

func TestSendWithoutAServer(t *testing.T) {
	if _, _, err := Send(socket(t), "step", 5); !errors.Is(err, ErrNoServer) {
		t.Fatalf("error = %v, want ErrNoServer", err)
	}
}

func TestServeReplacesADeadSocketButNotALiveOne(t *testing.T) {
	path := socket(t)
	dead, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	dead.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = dead.Close()

	handler := func(string, int) (int, int, error) { return 1, 2, nil }
	stop, err := Serve(path, handler)
	if err != nil {
		t.Fatalf("Serve over a dead socket: %v", err)
	}
	defer stop()

	if _, err := Serve(path, handler); err == nil {
		t.Fatal("a second server on a live socket must fail")
	}
}

func TestSendBadAnswers(t *testing.T) {
	path := socket(t)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		for _, reply := range []string{"what\n", ""} {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_, _ = conn.Write([]byte(reply))
			_ = conn.Close()
		}
	}()

	for range 2 {
		if _, _, err := Send(path, "step", 1); err == nil || errors.Is(err, ErrNoServer) {
			t.Fatalf("error = %v, want a bad answer error", err)
		}
	}
}

func TestPath(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	if got := filepath.Dir(Path()); got != "/run/user/1000" {
		t.Fatalf("socket dir = %q", got)
	}

	home := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "relative")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	dir := Dir()
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() || filepath.Base(dir) != "monctl" || info.Mode().Perm() != 0o700 {
		t.Fatalf("Dir = %q, %v, %v; want a private monctl directory", dir, info, err)
	}
}

func TestDirRefusesADirectoryOthersCanReplace(t *testing.T) {
	dir := t.TempDir()
	if !private(dir) {
		t.Fatal("a directory of this user must be usable")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if private(link) || private(filepath.Join(dir, "missing")) {
		t.Fatal("a link or a missing directory must be refused")
	}
}

func TestALongLineIsDropped(t *testing.T) {
	path := socket(t)
	stop, err := Serve(path, func(string, int) (int, int, error) {
		t.Error("the handler must not see a long line")
		return 0, 0, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_, _ = conn.Write(append(make([]byte, 4096), "\nset 1\n"...))
	if n, _ := conn.Read(make([]byte, 16)); n != 0 {
		t.Fatalf("got %d bytes back for a long line", n)
	}
}

func TestNoPathMeansNoServer(t *testing.T) {
	if _, err := Serve("", func(string, int) (int, int, error) { return 0, 0, nil }); err == nil {
		t.Fatal("Serve without a path must fail")
	}
	if _, _, err := Send("", "step", 1); !errors.Is(err, ErrNoServer) {
		t.Fatalf("error = %v, want ErrNoServer", err)
	}
}

func TestDirFallsBackToATempDirectoryOfThisUser(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("TMPDIR", tmp)

	if got, want := Dir(), filepath.Join(tmp, fmt.Sprintf("monctl-%d", os.Getuid())); got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}
}

func TestDirTightensALooseDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !private(dir) {
		t.Fatal("a loose directory of this user must be tightened and used")
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("mode = %v, %v; want 0700", info.Mode().Perm(), err)
	}
}
