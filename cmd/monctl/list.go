package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/utkudarilmaz/monctl/internal/profile"
)

type listEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	ProductCode  string `json:"productCode"`
	Connector    string `json:"connector"`
	Bus          string `json:"bus"`
	Profile      string `json:"profile"`
}

func (a app) list(opts options) int {
	if len(opts.args) > 0 {
		return a.usageError("list takes no value")
	}
	displays, err := a.detect()
	if err != nil {
		return a.fail(err)
	}
	all, err := a.profiles()
	if err != nil {
		return a.fail(err)
	}

	entries := make([]listEntry, 0, len(displays))
	for _, d := range displays {
		p, _ := profile.Find(all, d.Info)
		entries = append(entries, listEntry{
			ID:           d.ID,
			Name:         d.Info.Name,
			Manufacturer: d.Info.Manufacturer,
			ProductCode:  fmt.Sprintf("0x%04x", d.Info.ProductCode),
			Connector:    d.Connector,
			Bus:          d.Bus,
			Profile:      p.Name,
		})
	}

	if opts.json {
		return a.printJSON(entries)
	}
	if len(entries) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "No external monitor found.")
		return 0
	}

	w := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tCONNECTOR\tBUS\tPROFILE")
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ID, e.Name, e.Connector, e.Bus, orDash(e.Profile))
	}
	_ = w.Flush()
	return 0
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
