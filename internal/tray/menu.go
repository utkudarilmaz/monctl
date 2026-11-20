// Package tray shows monctl as a menu in the system tray.
package tray

import (
	"errors"
	"fmt"

	"github.com/utkudarilmaz/monctl/internal/ddc"
)

// Step is how much the brighter and dimmer items change the brightness.
const Step = 10

// Presets are the brightness levels offered in the menu, in percent.
var Presets = []int{0, 25, 50, 75, 100}

// ErrNoMonitor means no external monitor is attached.
var ErrNoMonitor = errors.New("tray: no external monitor found")

// Controller reads and changes the monitor for the menu.
type Controller interface {
	State() (State, error)
	SetBrightness(percent int) error
	AdjustBrightness(delta int) error
	SetInput(id string) error
}

// Input is one input the menu can switch to.
type Input struct {
	ID    string
	Label string
}

// State is what the monitor reports.
type State struct {
	Brightness int
	Max        int
	Current    string
	Inputs     []Input
}

// InputView is one input line of the menu.
type InputView struct {
	ID      string
	Title   string
	Current bool
}

// View is the content of the menu.
type View struct {
	Status  string
	Enabled bool
	Inputs  []InputView
}

// Render turns a monitor state, or the error from reading it, into the menu.
func Render(state State, err error) View {
	switch {
	case errors.Is(err, ErrNoMonitor):
		return View{Status: "No external monitor found"}
	case errors.Is(err, ddc.ErrNoReply):
		return View{Status: "Monitor is showing another computer"}
	case err != nil:
		return View{Status: "Cannot reach the monitor"}
	}

	view := View{Status: fmt.Sprintf("Brightness: %d%%", Percent(state.Brightness, state.Max)), Enabled: true}
	for _, in := range state.Inputs {
		view.Inputs = append(view.Inputs, InputView{ID: in.ID, Title: in.Label, Current: in.ID == state.Current})
	}
	return view
}

// Percent converts a raw brightness to percent of the monitor maximum.
func Percent(value, limit int) int {
	if limit <= 0 {
		return 0
	}
	return max(0, min(100, (value*100+limit/2)/limit))
}

// FromPercent converts percent to the raw brightness of the monitor.
func FromPercent(percent, limit int) int {
	percent = max(0, min(100, percent))
	return (percent*limit + 50) / 100
}
