// Package config reads the optional user settings of monctl.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/utkudarilmaz/monctl/internal/profile"
)

// FileName is the name of the settings file inside the config directory.
const FileName = "config.json"

// Config is the content of the settings file.
type Config struct {
	Inputs map[string]Input `json:"inputs"`
}

// Input changes how one input of the monitor profile is shown.
type Input struct {
	Label  string `json:"label"`
	Hidden bool   `json:"hidden"`
	// Before is a program and its arguments, run before the monitor is
	// switched to this input. The switch is skipped when it fails.
	Before []string `json:"before"`
}

// Dir returns the monctl config directory: $XDG_CONFIG_HOME/monctl, or
// ~/.config/monctl on every system.
func Dir() (string, error) {
	if base := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(base) {
		return filepath.Join(base, "monctl"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "monctl"), nil
}

// Load reads the settings file in dir. A missing file gives an empty Config.
func Load(dir string) (Config, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	for id, in := range c.Inputs {
		if !profile.Printable(in.Label) {
			return Config{}, fmt.Errorf("%s: input %q has a label with a character that cannot be printed", path, id)
		}
		for _, arg := range in.Before {
			if arg == "" || !profile.Printable(arg) {
				return Config{}, fmt.Errorf("%s: input %q has an empty or unprintable word in before", path, id)
			}
		}
	}
	return c, nil
}
