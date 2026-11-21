package keys

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	mediaKeys  = "org.gnome.settings-daemon.plugins.media-keys"
	customKey  = mediaKeys + ".custom-keybinding:"
	customList = "custom-keybindings"
	customRoot = "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/"
	step       = 5
)

type shortcut struct {
	id      string
	name    string
	binding string
	static  string
	delta   string
}

var shortcuts = []shortcut{
	{"monctl-brighter", "monctl brighter", "XF86MonBrightnessUp", "screen-brightness-up-static", fmt.Sprintf("+%d", step)},
	{"monctl-dimmer", "monctl dimmer", "XF86MonBrightnessDown", "screen-brightness-down-static", fmt.Sprintf("-%d", step)},
}

var quoted = regexp.MustCompile(`'([^']*)'`)

// settings runs the gsettings program with the given arguments.
type settings func(args ...string) (string, error)

// register adds the monctl shortcuts to the GNOME custom shortcuts. It does
// this once: the marker file keeps a shortcut the user removed from coming
// back. A computer with its own backlight keeps its brightness keys, and a
// key the user bound to something else is left alone.
func register(run settings, program, marker string, hasBacklight bool) error {
	if _, err := os.Stat(marker); err == nil || hasBacklight {
		return nil
	}
	out, err := run("get", mediaKeys, customList)
	if err != nil {
		return nil
	}
	paths := list(out)
	taken := map[string]bool{}
	for _, path := range paths {
		if binding, err := run("get", customKey+path, "binding"); err == nil {
			for _, key := range list(binding) {
				taken[key] = true
			}
		}
	}

	added := false
	for _, s := range shortcuts {
		path := customRoot + s.id + "/"
		if slices.Contains(paths, path) || taken[s.binding] {
			continue
		}
		for _, pair := range [][2]string{
			{"name", s.name},
			{"command", fmt.Sprintf("%s brightness %s", program, s.delta)},
			{"binding", s.binding},
		} {
			if _, err := run("set", customKey+path, pair[0], quote(pair[1])); err != nil {
				return fmt.Errorf("cannot bind the brightness keys: %w", err)
			}
		}
		if current, err := run("get", mediaKeys, s.static); err == nil && slices.Contains(list(current), s.binding) {
			_, _ = run("set", mediaKeys, s.static, "[]")
		}
		paths = append(paths, path)
		added = true
	}
	if added {
		if _, err := run("set", mediaKeys, customList, "['"+strings.Join(paths, "', '")+"']"); err != nil {
			return fmt.Errorf("cannot bind the brightness keys: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err != nil {
		return nil
	}
	return os.WriteFile(marker, nil, 0o600)
}

// quote writes text as a gsettings string value.
func quote(text string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(text) + "'"
}

// list reads the quoted items of a gsettings value such as ['a', 'b'].
func list(value string) []string {
	var items []string
	for _, m := range quoted.FindAllStringSubmatch(value, -1) {
		items = append(items, m[1])
	}
	return items
}
