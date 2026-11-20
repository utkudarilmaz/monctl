package tray

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/ddc"
)

func TestRenderWorkingMonitor(t *testing.T) {
	state := State{
		Brightness: 40, Max: 100, Current: "dp",
		Inputs: []Input{{ID: "dp", Label: "Workstation"}, {ID: "usbc", Label: "MacBook"}},
	}

	got := Render(state, nil)

	want := View{
		Status:  "Brightness: 40%",
		Enabled: true,
		Inputs: []InputView{
			{ID: "dp", Title: "Workstation", Current: true},
			{ID: "usbc", Title: "MacBook"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Render = %+v, want %+v", got, want)
	}
}

func TestRenderWithoutInputsOrKnownSource(t *testing.T) {
	got := Render(State{Brightness: 128, Max: 255}, nil)
	if got.Status != "Brightness: 50%" || !got.Enabled || len(got.Inputs) != 0 {
		t.Fatalf("Render = %+v", got)
	}
}

func TestRenderErrorsDisableTheMenu(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrNoMonitor, "No external monitor found"},
		{fmt.Errorf("wrapped: %w", ddc.ErrNoReply), "Monitor is showing another computer"},
		{errors.New("boom"), "Cannot reach the monitor"},
	}
	for _, tt := range tests {
		got := Render(State{Brightness: 40, Max: 100, Inputs: []Input{{ID: "dp"}}}, tt.err)
		if got.Status != tt.want || got.Enabled || len(got.Inputs) != 0 {
			t.Errorf("Render(%v) = %+v, want disabled with %q", tt.err, got, tt.want)
		}
	}
}

func TestPercent(t *testing.T) {
	tests := []struct{ value, limit, want int }{
		{0, 100, 0}, {40, 100, 40}, {100, 100, 100}, {150, 100, 100}, {-5, 100, 0},
		{128, 255, 50}, {255, 255, 100}, {1, 255, 0}, {2, 255, 1}, {10, 0, 0},
	}
	for _, tt := range tests {
		if got := Percent(tt.value, tt.limit); got != tt.want {
			t.Errorf("Percent(%d, %d) = %d, want %d", tt.value, tt.limit, got, tt.want)
		}
	}
}

func TestFromPercent(t *testing.T) {
	tests := []struct{ percent, limit, want int }{
		{0, 100, 0}, {25, 100, 25}, {100, 100, 100}, {120, 100, 100}, {-1, 100, 0},
		{50, 255, 128}, {100, 255, 255}, {25, 50, 13},
	}
	for _, tt := range tests {
		if got := FromPercent(tt.percent, tt.limit); got != tt.want {
			t.Errorf("FromPercent(%d, %d) = %d, want %d", tt.percent, tt.limit, got, tt.want)
		}
	}
}

func TestPresetsRoundTripOnA100Monitor(t *testing.T) {
	for _, p := range Presets {
		if got := Percent(FromPercent(p, 100), 100); got != p {
			t.Errorf("preset %d came back as %d", p, got)
		}
	}
}

func TestFailedMarksOnlyAWorkingMenu(t *testing.T) {
	ok := Render(State{Brightness: 50, Max: 100}, nil).Failed()
	if ok.Status != "Brightness: 50%, last change failed" {
		t.Fatalf("Status = %q", ok.Status)
	}
	down := Render(State{}, ErrNoMonitor)
	if got := down.Failed(); got.Status != down.Status {
		t.Fatalf("Status = %q, want %q", got.Status, down.Status)
	}
}

func TestOnceReportsAnErrorUntilItChanges(t *testing.T) {
	var got []string
	report := once(func(err error) { got = append(got, err.Error()) })

	for _, err := range []error{errors.New("a"), errors.New("a"), nil, errors.New("a"), errors.New("b"), nil} {
		report(err)
	}
	if want := []string{"a", "a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reported %v, want %v", got, want)
	}
}
