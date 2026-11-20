// Package profile describes what monctl knows about one monitor model.
package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/utkudarilmaz/monctl/internal/edid"
)

var (
	productCode = regexp.MustCompile(`^0x[0-9a-f]{4}$`)
	inputID     = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
)

// Profile is the data for one monitor model.
type Profile struct {
	Name   string  `json:"name"`
	Match  Match   `json:"match"`
	Inputs []Input `json:"inputs"`
}

// Match selects the monitors a profile applies to, by the fields of their EDID.
type Match struct {
	Manufacturer string `json:"manufacturer"`
	ProductCode  string `json:"productCode"`
}

// Input is one video port. Some monitors report a different value than the
// one that selects the port, so both are kept.
type Input struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Write uint16 `json:"write"`
	Read  uint16 `json:"read"`
}

// Parse reads and checks one profile.
func Parse(data []byte) (Profile, error) {
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, err
	}
	if err := p.validate(); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (p Profile) validate() error {
	if p.Name == "" {
		return errors.New("name is empty")
	}
	if !Printable(p.Name) {
		return errors.New("name has a character that cannot be printed")
	}
	if len(p.Match.Manufacturer) != 3 {
		return errors.New("match.manufacturer must be the three letter maker id")
	}
	if !productCode.MatchString(p.Match.ProductCode) {
		return errors.New(`match.productCode must look like "0x3cf0"`)
	}
	seen := map[string]bool{}
	for _, in := range p.Inputs {
		if !inputID.MatchString(in.ID) {
			return fmt.Errorf("input id %q must be lower case letters and digits", in.ID)
		}
		if seen[in.ID] {
			return fmt.Errorf("input id %q is used twice", in.ID)
		}
		seen[in.ID] = true
		if in.Label == "" {
			return fmt.Errorf("input %q has no label", in.ID)
		}
		if !Printable(in.Label) {
			return fmt.Errorf("input %q has a label with a character that cannot be printed", in.ID)
		}
		if in.Write > 0xff || in.Read > 0xff {
			return fmt.Errorf("input %q has a value above 255", in.ID)
		}
	}
	return nil
}

// Printable reports whether text is safe to show in a terminal. Control and
// format characters are refused because they can move the cursor or hide text.
func Printable(text string) bool {
	return strings.IndexFunc(text, func(r rune) bool { return !unicode.IsPrint(r) }) < 0
}

// Matches reports whether the profile applies to a monitor.
func (p Profile) Matches(info edid.Info) bool {
	return strings.EqualFold(p.Match.Manufacturer, info.Manufacturer) &&
		p.Match.ProductCode == fmt.Sprintf("0x%04x", info.ProductCode)
}

// Load reads every .json profile at the top of each file system. Earlier file
// systems win when two profiles match the same monitor. A nil file system is
// skipped.
func Load(sources ...fs.FS) ([]Profile, error) {
	var all []Profile
	for _, src := range sources {
		if src == nil {
			continue
		}
		names, err := fs.Glob(src, "*.json")
		if err != nil {
			return nil, err
		}
		sort.Strings(names)
		for _, name := range names {
			data, err := fs.ReadFile(src, name)
			if err != nil {
				return nil, err
			}
			p, err := Parse(data)
			if err != nil {
				return nil, fmt.Errorf("profile %q: %w", name, err)
			}
			all = append(all, p)
		}
	}
	return all, nil
}

// Find returns the first profile that applies to a monitor.
func Find(profiles []Profile, info edid.Info) (Profile, bool) {
	for _, p := range profiles {
		if p.Matches(info) {
			return p, true
		}
	}
	return Profile{}, false
}
