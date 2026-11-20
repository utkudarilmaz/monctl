package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Inputs) != 0 {
		t.Fatalf("unexpected config %+v", c)
	}
}

func TestLoadReadsInputs(t *testing.T) {
	dir := write(t, `{"inputs": {"dp": {"label": "Workstation"}, "hdmi1": {"hidden": true}}}`)

	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Inputs["dp"].Label != "Workstation" || !c.Inputs["hdmi1"].Hidden {
		t.Fatalf("unexpected config %+v", c)
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	for name, content := range map[string]string{
		"not json":      `{`,
		"unknown field": `{"input": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(write(t, content))
			if err == nil || !strings.Contains(err.Error(), FileName) {
				t.Fatalf("error = %v, want it to name the file", err)
			}
		})
	}
}

func TestDir(t *testing.T) {
	t.Setenv("HOME", "/home/someone")

	t.Setenv("XDG_CONFIG_HOME", "/custom")
	if got, _ := Dir(); got != "/custom/monctl" {
		t.Fatalf("Dir = %q, want /custom/monctl", got)
	}

	for _, ignored := range []string{"", "relative/path"} {
		t.Setenv("XDG_CONFIG_HOME", ignored)
		if got, _ := Dir(); got != "/home/someone/.config/monctl" {
			t.Fatalf("Dir with XDG_CONFIG_HOME=%q = %q", ignored, got)
		}
	}
}
