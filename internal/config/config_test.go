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
		"escape label":  `{"inputs": {"dp": {"label": "a\u001b[2Jb"}}}`,
		"tab in label":  `{"inputs": {"dp": {"label": "a\tb"}}}`,
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

func TestDirWithoutAHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if dir, err := Dir(); err == nil {
		t.Fatalf("Dir = %q, want an error", dir)
	}
}

func TestLoadReportsReadErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, FileName), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("a config path that is a directory gave no error")
	}
}

func TestLoadAcceptsEmptyAndNullInputs(t *testing.T) {
	for _, content := range []string{`{}`, `{"inputs": null}`, `{"inputs": {"dp": {}}}`} {
		c, err := Load(write(t, content))
		if err != nil {
			t.Errorf("Load(%s) error = %v", content, err)
			continue
		}
		if in := c.Inputs["dp"]; in.Label != "" || in.Hidden || in.Before != nil {
			t.Errorf("Load(%s) = %+v", content, c)
		}
	}
}

func TestLoadReadsBefore(t *testing.T) {
	dir := write(t, `{"inputs": {"dp": {"before": ["ssh", "workstation", "monctl", "wake"]}}}`)

	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Inputs["dp"].Before; len(got) != 4 || got[0] != "ssh" || got[3] != "wake" {
		t.Fatalf("before = %v", got)
	}
}

func TestLoadRejectsBadBefore(t *testing.T) {
	for name, content := range map[string]string{
		"empty word":   `{"inputs": {"dp": {"before": ["ssh", ""]}}}`,
		"control char": `{"inputs": {"dp": {"before": ["ssh", "a\u001bb"]}}}`,
		"not a list":   `{"inputs": {"dp": {"before": "ssh workstation"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, content)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
