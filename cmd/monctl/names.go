package main

import (
	"fmt"
	"maps"
	"strings"

	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/monitor"
)

// relabel builds the monitor with the names monctl has learned. A label from
// the settings file wins over a learned name. A learned name can come from
// another computer, so one that reads like another input is left out.
func (s *session) relabel() {
	cfg := s.config
	if s.profile != nil {
		cfg.Inputs = map[string]config.Input{}
		maps.Copy(cfg.Inputs, s.config.Inputs)
		taken := map[string]bool{}
		for _, in := range s.profile.Inputs {
			label := in.Label
			if user := s.config.Inputs[in.ID].Label; user != "" {
				label = user
			}
			taken[strings.ToLower(in.ID)], taken[strings.ToLower(label)] = true, true
		}
		for id, name := range s.names.Of(s.profile.Name) {
			if in := cfg.Inputs[id]; in.Label == "" && !taken[strings.ToLower(name)] {
				in.Label = name
				cfg.Inputs[id] = in
			}
		}
	}
	s.monitor = monitor.New(s.client, s.profile, cfg)
}

// source reads the input the monitor shows. A monitor only answers the
// computer it shows, so that input is this computer and gets its name.
func (a app) source(s *session) (monitor.Source, error) {
	src, err := s.monitor.Source()
	if err != nil || !src.Known || a.state == "" {
		return src, err
	}
	if a.remember(s, src.Input.ID, a.host()) {
		if in, err := s.monitor.Lookup(src.Input.ID); err == nil {
			src.Input = in
		}
	}
	return src, nil
}

// remember stores the computer name of an input and reports whether the
// labels changed.
func (a app) remember(s *session, id, name string) bool {
	if a.state == "" || s.profile == nil || !s.names.Set(s.profile.Name, id, name) {
		return false
	}
	if err := s.names.Save(a.state); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "monctl: cannot remember the computer names: %v\n", err)
	}
	s.relabel()
	return true
}
