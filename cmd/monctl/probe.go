package main

import (
	"errors"
	"fmt"

	"github.com/utkudarilmaz/monctl/internal/ddc"
)

type probeOutput struct {
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	ProductCode  string `json:"productCode"`
	Profile      string `json:"profile"`
	Brightness   string `json:"brightness"`
	InputValue   string `json:"inputValue"`
	Capabilities string `json:"capabilities"`
}

// probe leaves the serial number out, so its output can be posted in public.
func (a app) probe(opts options) int {
	if len(opts.args) > 0 {
		return a.usageError("probe takes no value")
	}
	s, err := a.connect(opts.display)
	if err != nil {
		return a.fail(err)
	}
	defer s.close()

	out := probeOutput{
		Name:         s.display.Info.Name,
		Manufacturer: s.display.Info.Manufacturer,
		ProductCode:  fmt.Sprintf("0x%04x", s.display.Info.ProductCode),
		Profile:      "none",
	}
	if s.profile != nil {
		out.Profile = s.profile.Name
	}

	var silent bool
	report := func(text string, err error) string {
		if err != nil {
			silent = silent || errors.Is(err, ddc.ErrNoReply)
			return "failed: " + err.Error()
		}
		return text
	}

	level, err := s.client.GetVCP(0x10)
	out.Brightness = report(fmt.Sprintf("%d (max %d)", level.Current, level.Max), err)
	source, err := s.client.GetVCP(0x60)
	out.InputValue = report(fmt.Sprintf("%d (max %d)", source.Current&0xff, source.Max), err)
	caps, err := s.client.Capabilities()
	out.Capabilities = report(caps, err)

	if opts.json {
		return a.printJSON(out)
	}
	_, _ = fmt.Fprintf(a.stdout, "Monitor:       %s\n", out.Name)
	_, _ = fmt.Fprintf(a.stdout, "Manufacturer:  %s\n", out.Manufacturer)
	_, _ = fmt.Fprintf(a.stdout, "Product code:  %s\n", out.ProductCode)
	_, _ = fmt.Fprintf(a.stdout, "Profile:       %s\n", out.Profile)
	_, _ = fmt.Fprintf(a.stdout, "Brightness:    %s\n", out.Brightness)
	_, _ = fmt.Fprintf(a.stdout, "Input value:   %s\n", out.InputValue)
	_, _ = fmt.Fprintf(a.stdout, "Capabilities:  %s\n", out.Capabilities)
	if silent {
		_, _ = fmt.Fprint(a.stderr, "\n"+silentHelp)
		return 1
	}
	return 0
}
