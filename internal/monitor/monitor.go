// Package monitor turns DDC/CI features into the settings monctl offers.
package monitor

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/profile"
)

const (
	vcpBrightness = 0x10
	vcpInput      = 0x60
)

var (
	// ErrNoProfile means monctl has no input list for this monitor model.
	ErrNoProfile = errors.New("monitor: no profile for this monitor")
	// ErrUnknownInput means the name is not an input of this monitor.
	ErrUnknownInput = errors.New("monitor: unknown input")
)

// Control is the part of the DDC/CI client a Monitor needs.
type Control interface {
	GetVCP(code byte) (ddc.Value, error)
	SetVCP(code byte, value uint16) error
}

// Input is one video port of the monitor, with the user's label applied.
type Input struct {
	ID     string
	Label  string
	Hidden bool

	write uint16
	read  uint16
}

// Level is a brightness reading.
type Level struct {
	Current int
	Max     int
}

// Source is the input the monitor shows now. Known is false when the value
// is not in the profile.
type Source struct {
	Value uint16
	Input Input
	Known bool
}

// Monitor is one display with its profile and user settings.
type Monitor struct {
	control    Control
	inputs     []Input
	hasProfile bool
}

// New builds a Monitor. prof is nil when no profile matches the display.
func New(control Control, prof *profile.Profile, cfg config.Config) *Monitor {
	m := &Monitor{control: control, hasProfile: prof != nil}
	if prof == nil {
		return m
	}
	for _, in := range prof.Inputs {
		input := Input{ID: in.ID, Label: in.Label, write: in.Write, read: in.Read}
		if user, ok := cfg.Inputs[in.ID]; ok {
			if user.Label != "" {
				input.Label = user.Label
			}
			input.Hidden = user.Hidden
		}
		m.inputs = append(m.inputs, input)
	}
	return m
}

// HasProfile reports whether the monitor model is known.
func (m *Monitor) HasProfile() bool { return m.hasProfile }

// Inputs lists the inputs the user wants to see.
func (m *Monitor) Inputs() []Input {
	var visible []Input
	for _, in := range m.inputs {
		if !in.Hidden {
			visible = append(visible, in)
		}
	}
	return visible
}

// Brightness reads the current brightness.
func (m *Monitor) Brightness() (Level, error) {
	v, err := m.control.GetVCP(vcpBrightness)
	if err != nil {
		return Level{}, err
	}
	level := Level{Current: int(v.Current), Max: int(v.Max)}
	if level.Max == 0 {
		level.Max = 100
	}
	return level, nil
}

// SetBrightness sets the brightness, kept between 0 and the monitor's
// maximum, and returns the value it sent. It is not read back, which keeps a
// change short enough for repeated key presses.
func (m *Monitor) SetBrightness(value int) (Level, error) {
	level, err := m.Brightness()
	if err != nil {
		return Level{}, err
	}
	return m.writeBrightness(value, level.Max)
}

// WriteBrightness sends a raw brightness without reading the monitor first.
// The caller keeps the value inside the monitor's range.
func (m *Monitor) WriteBrightness(value int) error {
	return m.control.SetVCP(vcpBrightness, uint16(max(0, min(value, 0xffff))))
}

// AdjustBrightness changes the brightness by delta.
func (m *Monitor) AdjustBrightness(delta int) (Level, error) {
	level, err := m.Brightness()
	if err != nil {
		return Level{}, err
	}
	delta = max(-math.MaxUint16, min(delta, math.MaxUint16))
	return m.writeBrightness(level.Current+delta, level.Max)
}

func (m *Monitor) writeBrightness(value, limit int) (Level, error) {
	value = max(0, min(value, limit))
	if err := m.control.SetVCP(vcpBrightness, uint16(value)); err != nil {
		return Level{}, err
	}
	return Level{Current: value, Max: limit}, nil
}

// Source reads the input the monitor shows now.
func (m *Monitor) Source() (Source, error) {
	v, err := m.control.GetVCP(vcpInput)
	if err != nil {
		return Source{}, err
	}
	src := Source{Value: v.Current & 0xff}
	for _, in := range m.inputs {
		if in.read == src.Value {
			src.Input, src.Known = in, true
			break
		}
	}
	return src, nil
}

// SetInput switches to the input with the given id or label. The monitor
// stops answering once it shows another computer, so nothing is read back.
func (m *Monitor) SetInput(name string) (Input, error) {
	in, err := m.Lookup(name)
	if err != nil {
		return Input{}, err
	}
	return in, m.control.SetVCP(vcpInput, in.write)
}

// Lookup finds the input with the given id or label without switching to it.
func (m *Monitor) Lookup(name string) (Input, error) {
	if !m.hasProfile {
		return Input{}, ErrNoProfile
	}
	in, ok := m.find(name)
	if !ok {
		return Input{}, fmt.Errorf("%w %q", ErrUnknownInput, name)
	}
	return in, nil
}

// SetInputRaw writes an input value that is not checked against a profile.
func (m *Monitor) SetInputRaw(value uint8) error {
	return m.control.SetVCP(vcpInput, uint16(value))
}

func (m *Monitor) find(name string) (Input, bool) {
	for _, in := range m.inputs {
		if strings.EqualFold(in.ID, name) {
			return in, true
		}
	}
	for _, in := range m.inputs {
		if strings.EqualFold(in.Label, name) {
			return in, true
		}
	}
	return Input{}, false
}
