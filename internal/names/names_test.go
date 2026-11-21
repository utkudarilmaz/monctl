package names

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetMovesAComputerToItsNewInput(t *testing.T) {
	var b Book
	if !b.Set("m", "dp", "desk") {
		t.Fatal("a new name was not reported as new")
	}
	if b.Set("m", "dp", "desk") {
		t.Fatal("the same name was reported as new")
	}
	b.Set("m", "usbc", "laptop")
	b.Set("m", "usbc", "desk")

	got := b.Of("m")
	if len(got) != 1 || got["usbc"] != "desk" {
		t.Fatalf("names are %v, want only usbc=desk", got)
	}
}

func TestSetIgnoresAnEmptyName(t *testing.T) {
	var b Book
	if b.Set("m", "dp", " \x07 ") || len(b.Monitors) != 0 {
		t.Fatal("an empty name was stored")
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "names.json")
	var b Book
	b.Set("m", "dp", "desk")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	if got := Load(path).Of("m")["dp"]; got != "desk" {
		t.Fatalf("loaded %q, want desk", got)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode is %v, want 0600", info.Mode().Perm())
	}
}

func TestLoadCleansWhatTheFileHolds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "names.json")
	if err := os.WriteFile(path, []byte(`{"monitors":{"m":{"dp":"de\u001bsk","usbc":"\u0007"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Load(path).Of("m")
	if got["dp"] != "desk" || len(got) != 1 {
		t.Fatalf("loaded %v, want only dp=desk", got)
	}
}

func TestLoadOfAMissingOrBrokenFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	_ = os.WriteFile(broken, []byte("{"), 0o600)
	for _, path := range []string{filepath.Join(dir, "none.json"), broken, ""} {
		if b := Load(path); len(b.Monitors) != 0 {
			t.Fatalf("Load(%q) = %v, want empty", path, b)
		}
	}
}

func TestParse(t *testing.T) {
	cases := map[string]string{
		"computer: desk\n":                      "desk",
		"warning\ncomputer: My Laptop\nmore\n":  "My Laptop",
		"computer: a\ncomputer: b":              "b",
		"no name here":                          "",
		"computer: bad\x1b[31mname":             "bad[31mname",
		"computer: " + strings.Repeat("x", 200): strings.Repeat("x", 64),
	}
	for output, want := range cases {
		if got := Parse(output); got != want {
			t.Errorf("Parse(%q) = %q, want %q", output, got, want)
		}
	}
}

func TestLineIsReadBackByParse(t *testing.T) {
	if got := Parse(Line("desk")); got != "desk" {
		t.Fatalf("got %q", got)
	}
}

func TestPathUsesTheStateDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/somewhere/state")
	if got := Path(); got != "/somewhere/state/monctl/names.json" {
		t.Fatalf("Path() = %q", got)
	}
}

func TestPathFallsBackToTheHomeDirectory(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "relative")
	t.Setenv("HOME", "/home/someone")
	if got := Path(); got != "/home/someone/.local/state/monctl/names.json" {
		t.Fatalf("Path() = %q", got)
	}
}

func TestPathIsEmptyWithoutAHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if got := Path(); got != "" {
		t.Fatalf("Path() = %q, want empty", got)
	}
}

func TestSaveFailsWhenTheDirectoryCannotBeMade(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var b Book
	b.Set("m", "dp", "desk")
	if err := b.Save(filepath.Join(blocker, "names.json")); err == nil {
		t.Fatal("Save under a file must fail")
	}
}

func TestHostIsTheShortHostName(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS uses the computer name")
	}
	host, err := os.Hostname()
	if err != nil {
		t.Skip(err)
	}
	short, _, _ := strings.Cut(host, ".")
	if got := Host(); got != Clean(short) {
		t.Fatalf("Host() = %q, want %q", got, Clean(short))
	}
}
