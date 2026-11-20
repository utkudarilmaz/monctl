package profile

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/utkudarilmaz/monctl/internal/edid"
	"github.com/utkudarilmaz/monctl/profiles"
)

const valid = `{
  "name": "Test Monitor",
  "match": {"manufacturer": "TST", "productCode": "0x00ab"},
  "inputs": [{"id": "dp", "label": "DisplayPort", "write": 5, "read": 7}]
}`

func TestParseValid(t *testing.T) {
	p, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Test Monitor" || len(p.Inputs) != 1 {
		t.Fatalf("unexpected profile %+v", p)
	}
	if in := p.Inputs[0]; in.ID != "dp" || in.Write != 5 || in.Read != 7 {
		t.Fatalf("unexpected input %+v", in)
	}
}

func TestParseRejectsBadProfiles(t *testing.T) {
	tests := map[string]string{
		"not json":          `{`,
		"no name":           `{"match":{"manufacturer":"TST","productCode":"0x00ab"}}`,
		"short maker":       `{"name":"x","match":{"manufacturer":"TS","productCode":"0x00ab"}}`,
		"decimal product":   `{"name":"x","match":{"manufacturer":"TST","productCode":"171"}}`,
		"upper hex product": `{"name":"x","match":{"manufacturer":"TST","productCode":"0x00AB"}}`,
		"bad input id":      `{"name":"x","match":{"manufacturer":"TST","productCode":"0x00ab"},"inputs":[{"id":"DP 1","label":"a"}]}`,
		"duplicate input":   `{"name":"x","match":{"manufacturer":"TST","productCode":"0x00ab"},"inputs":[{"id":"dp","label":"a"},{"id":"dp","label":"b"}]}`,
		"no label":          `{"name":"x","match":{"manufacturer":"TST","productCode":"0x00ab"},"inputs":[{"id":"dp"}]}`,
		"value too large":   `{"name":"x","match":{"manufacturer":"TST","productCode":"0x00ab"},"inputs":[{"id":"dp","label":"a","write":256}]}`,
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestMatches(t *testing.T) {
	p, _ := Parse([]byte(valid))
	tests := []struct {
		info edid.Info
		want bool
	}{
		{edid.Info{Manufacturer: "TST", ProductCode: 0x00ab}, true},
		{edid.Info{Manufacturer: "tst", ProductCode: 0x00ab}, true},
		{edid.Info{Manufacturer: "TST", ProductCode: 0x00ac}, false},
		{edid.Info{Manufacturer: "XYZ", ProductCode: 0x00ab}, false},
	}
	for _, tt := range tests {
		if got := p.Matches(tt.info); got != tt.want {
			t.Errorf("Matches(%+v) = %v, want %v", tt.info, got, tt.want)
		}
	}
}

func TestLoadPrefersEarlierSources(t *testing.T) {
	user := fstest.MapFS{"mine.json": {Data: []byte(strings.Replace(valid, "Test Monitor", "Mine", 1))}}
	shipped := fstest.MapFS{
		"a.json":     {Data: []byte(valid)},
		"notes.txt":  {Data: []byte("ignored")},
		"sub/b.json": {Data: []byte("{")},
	}

	all, err := Load(user, nil, shipped)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("loaded %d profiles, want 2", len(all))
	}
	p, ok := Find(all, edid.Info{Manufacturer: "TST", ProductCode: 0x00ab})
	if !ok || p.Name != "Mine" {
		t.Fatalf("Find = %+v, %v, want the user profile", p, ok)
	}
	if _, ok := Find(all, edid.Info{Manufacturer: "XYZ"}); ok {
		t.Fatal("found a profile for an unknown monitor")
	}
}

func TestLoadNamesTheBrokenFile(t *testing.T) {
	_, err := Load(fstest.MapFS{"broken.json": {Data: []byte("{")}})
	if err == nil || !strings.Contains(err.Error(), "broken.json") {
		t.Fatalf("error = %v, want it to name broken.json", err)
	}
}

func TestShippedProfilesAreValid(t *testing.T) {
	all, err := Load(profiles.FS)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := Find(all, edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0})
	if !ok {
		t.Fatal("no profile for the MSI MPG 274URDFW E16M")
	}
	want := map[string][2]uint16{"dp": {5, 7}, "usbc": {8, 8}}
	for _, in := range p.Inputs {
		if w, ok := want[in.ID]; !ok || in.Write != w[0] || in.Read != w[1] {
			t.Errorf("unexpected input %+v", in)
		}
	}
}
