package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/profile"
)

var testProfile = &profile.Profile{
	Name: "Test",
	Inputs: []profile.Input{
		{ID: "dp", Label: "DisplayPort", Write: 5, Read: 7},
		{ID: "usbc", Label: "USB-C", Write: 8, Read: 8},
		{ID: "hdmi1", Label: "HDMI 1", Write: 1, Read: 1},
	},
}

func newMonitor(prof *profile.Profile, cfg config.Config) (*Monitor, *bustest.Fake) {
	fake := bustest.NewFake()
	fake.VCP[0x10] = bustest.VCP{Current: 50, Max: 100}
	fake.VCP[0x60] = bustest.VCP{Current: 0x0f07, Max: 14}
	client := ddc.New(fake, ddc.WithSleep(func(time.Duration) {}))
	return New(client, prof, cfg), fake
}

func TestBrightness(t *testing.T) {
	m, _ := newMonitor(testProfile, config.Config{})

	level, err := m.Brightness()
	if err != nil {
		t.Fatal(err)
	}
	if level != (Level{Current: 50, Max: 100}) {
		t.Fatalf("level = %+v", level)
	}
}

func TestBrightnessAssumesMax100WhenTheMonitorSaysZero(t *testing.T) {
	m, fake := newMonitor(testProfile, config.Config{})
	fake.VCP[0x10] = bustest.VCP{Current: 30}

	level, _ := m.Brightness()
	if level.Max != 100 {
		t.Fatalf("max = %d, want 100", level.Max)
	}
}

func TestSetBrightnessClamps(t *testing.T) {
	tests := []struct{ set, want int }{{40, 40}, {0, 0}, {100, 100}, {250, 100}, {-5, 0}}
	for _, tt := range tests {
		m, fake := newMonitor(testProfile, config.Config{})
		level, err := m.SetBrightness(tt.set)
		if err != nil {
			t.Fatal(err)
		}
		if level.Current != tt.want || int(fake.VCP[0x10].Current) != tt.want {
			t.Errorf("SetBrightness(%d) = %d, monitor has %d, want %d", tt.set, level.Current, fake.VCP[0x10].Current, tt.want)
		}
	}
}

func TestAdjustBrightnessClamps(t *testing.T) {
	tests := []struct{ delta, want int }{{10, 60}, {-10, 40}, {70, 100}, {-70, 0}}
	for _, tt := range tests {
		m, _ := newMonitor(testProfile, config.Config{})
		level, err := m.AdjustBrightness(tt.delta)
		if err != nil {
			t.Fatal(err)
		}
		if level.Current != tt.want {
			t.Errorf("AdjustBrightness(%d) = %d, want %d", tt.delta, level.Current, tt.want)
		}
	}
}

func TestBrightnessErrorsWhenTheMonitorIsSilent(t *testing.T) {
	m, fake := newMonitor(testProfile, config.Config{})
	fake.Asleep = true

	if _, err := m.Brightness(); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("Brightness error = %v, want ErrNoReply", err)
	}
	if _, err := m.SetBrightness(10); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("SetBrightness error = %v, want ErrNoReply", err)
	}
	if _, err := m.AdjustBrightness(10); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("AdjustBrightness error = %v, want ErrNoReply", err)
	}
	if _, err := m.Source(); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("Source error = %v, want ErrNoReply", err)
	}
}

func TestSourceUsesTheReadValueAndTheLowByte(t *testing.T) {
	m, _ := newMonitor(testProfile, config.Config{})

	src, err := m.Source()
	if err != nil {
		t.Fatal(err)
	}
	if !src.Known || src.Input.ID != "dp" || src.Value != 7 {
		t.Fatalf("source = %+v, want dp with value 7", src)
	}
}

func TestSourceUnknownValue(t *testing.T) {
	m, fake := newMonitor(testProfile, config.Config{})
	fake.VCP[0x60] = bustest.VCP{Current: 12}

	src, err := m.Source()
	if err != nil {
		t.Fatal(err)
	}
	if src.Known || src.Value != 12 {
		t.Fatalf("source = %+v, want unknown value 12", src)
	}
}

func TestSetInputWritesTheWriteValue(t *testing.T) {
	for _, name := range []string{"dp", "DP", "displayport"} {
		m, fake := newMonitor(testProfile, config.Config{})
		in, err := m.SetInput(name)
		if err != nil {
			t.Fatal(err)
		}
		if in.ID != "dp" || fake.VCP[0x60].Current != 5 {
			t.Errorf("SetInput(%q) chose %q and wrote %d, want dp and 5", name, in.ID, fake.VCP[0x60].Current)
		}
	}
}

func TestSetInputByUserLabelAndHiddenID(t *testing.T) {
	cfg := config.Config{Inputs: map[string]config.Input{
		"usbc":  {Label: "MacBook"},
		"hdmi1": {Hidden: true},
	}}
	m, fake := newMonitor(testProfile, cfg)

	if in, err := m.SetInput("macbook"); err != nil || in.ID != "usbc" || in.Label != "MacBook" {
		t.Fatalf("SetInput(macbook) = %+v, %v", in, err)
	}
	if _, err := m.SetInput("hdmi1"); err != nil || fake.VCP[0x60].Current != 1 {
		t.Fatalf("a hidden input must still work by id: %v", err)
	}

	visible := m.Inputs()
	if len(visible) != 2 || visible[0].ID != "dp" || visible[1].Label != "MacBook" {
		t.Fatalf("Inputs = %+v, want dp and MacBook", visible)
	}
}

func TestSetInputRefusesUnknownNames(t *testing.T) {
	m, fake := newMonitor(testProfile, config.Config{})
	before := len(fake.Writes)

	if _, err := m.SetInput("vga"); !errors.Is(err, ErrUnknownInput) {
		t.Fatalf("error = %v, want ErrUnknownInput", err)
	}
	if len(fake.Writes) != before {
		t.Fatal("nothing must be written for an unknown input")
	}
}

func TestNoProfile(t *testing.T) {
	m, fake := newMonitor(nil, config.Config{})

	if m.HasProfile() || len(m.Inputs()) != 0 {
		t.Fatal("a monitor without a profile has no inputs")
	}
	if _, err := m.SetInput("dp"); !errors.Is(err, ErrNoProfile) {
		t.Fatalf("error = %v, want ErrNoProfile", err)
	}
	if src, err := m.Source(); err != nil || src.Known || src.Value != 7 {
		t.Fatalf("Source = %+v, %v", src, err)
	}
	if err := m.SetInputRaw(15); err != nil || fake.VCP[0x60].Current != 15 {
		t.Fatalf("SetInputRaw did not write 15: %v", err)
	}
	if level, err := m.SetBrightness(20); err != nil || level.Current != 20 {
		t.Fatalf("brightness must work without a profile: %+v, %v", level, err)
	}
}
