// Package names remembers which computer sits on which input of a monitor.
package names

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	fileName = "names.json"
	marker   = "computer: "
	maxRunes = 64
)

// Book holds the computer name of each input, per monitor model.
type Book struct {
	Monitors map[string]map[string]string `json:"monitors"`
}

// Path returns the file the names are kept in: $XDG_STATE_HOME/monctl, or
// ~/.local/state/monctl on every system. It is empty when there is no home.
func Path() string {
	if base := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(base) {
		return filepath.Join(base, "monctl", fileName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "state", "monctl", fileName)
}

// Load reads the names file. A missing or broken file gives an empty Book.
func Load(path string) Book {
	var b Book
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &b)
	}
	for _, inputs := range b.Monitors {
		for id, name := range inputs {
			if clean := Clean(name); clean == "" {
				delete(inputs, id)
			} else {
				inputs[id] = clean
			}
		}
	}
	return b
}

// Save writes the names file, readable by this user only.
func (b Book) Save(path string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Of returns the names known for the inputs of one monitor model.
func (b Book) Of(monitor string) map[string]string { return b.Monitors[monitor] }

// Set records that a computer sits on an input, and reports whether that is
// new. A computer sits on one input only, so its name leaves the others.
func (b *Book) Set(monitor, input, name string) bool {
	name = Clean(name)
	if name == "" || b.Monitors[monitor][input] == name {
		return false
	}
	if b.Monitors == nil {
		b.Monitors = map[string]map[string]string{}
	}
	if b.Monitors[monitor] == nil {
		b.Monitors[monitor] = map[string]string{}
	}
	for id, old := range b.Monitors[monitor] {
		if old == name {
			delete(b.Monitors[monitor], id)
		}
	}
	b.Monitors[monitor][input] = name
	return true
}

// Line is what a computer prints to tell its name.
func Line(name string) string { return marker + name }

// Parse finds the name a computer printed in the output of a command. The
// output may come from another computer, so the name is cleaned.
func Parse(output string) string {
	found := ""
	for _, line := range strings.Split(output, "\n") {
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), marker); ok {
			found = Clean(name)
		}
	}
	return found
}

// Clean makes a name safe to show: no control characters, no long text.
func Clean(name string) string {
	runes := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, name)))
	return strings.TrimSpace(string(runes[:min(len(runes), maxRunes)]))
}
