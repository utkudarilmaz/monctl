package display

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
)

type services struct {
	fakes   map[uint64]*bustest.Fake
	order   []uint64
	listErr error
}

func (s *services) add(id uint64, edid []byte) *bustest.Fake {
	if s.fakes == nil {
		s.fakes = map[uint64]*bustest.Fake{}
	}
	f := bustest.NewFake()
	f.EDID = edid
	s.fakes[id] = f
	s.order = append(s.order, id)
	return f
}

func (s *services) detector() detector {
	return detector{
		list: func() ([]uint64, error) { return s.order, s.listErr },
		open: func(id uint64) (bus.Bus, error) {
			f, ok := s.fakes[id]
			if !ok {
				return nil, bus.ErrNoAVService
			}
			return f, nil
		},
	}
}

func TestDetectFindsExternalDisplays(t *testing.T) {
	var s services
	first := s.add(4295167419, msiEDID(t))
	s.add(77, msiEDID(t))

	got, err := s.detector().detect()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("found %d displays, want 2", len(got))
	}
	if got[0].Bus != "avservice:4295167419" || got[0].Info.Name != "MPG 274U E16M" {
		t.Fatalf("unexpected display %+v", got[0])
	}
	if got[0].ID == got[1].ID || got[0].ID == "" {
		t.Fatalf("ids %q and %q must differ", got[0].ID, got[1].ID)
	}
	if !first.Closed {
		t.Fatal("the service was not closed after reading the EDID")
	}
}

func TestDetectSkipsServicesWithoutAValidEDID(t *testing.T) {
	var s services
	s.add(1, make([]byte, 128))
	s.add(2, msiEDID(t)).FailReads = 1
	s.order = append(s.order, 3)
	s.add(4, msiEDID(t))

	got, err := s.detector().detect()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Bus != "avservice:4" {
		t.Fatalf("got %+v, want only service 4", got)
	}
}

func TestDetectNothingAttached(t *testing.T) {
	var s services
	got, err := s.detector().detect()
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing", got, err)
	}
}

func TestDetectListError(t *testing.T) {
	s := services{listErr: errors.New("boom")}
	if _, err := s.detector().detect(); err == nil {
		t.Fatal("expected an error")
	}
}

func TestOpenDisplay(t *testing.T) {
	var s services
	want := s.add(42, msiEDID(t))

	b, err := s.detector().openDisplay(Display{ID: "msi-3cf0", Bus: "avservice:42"})
	if err != nil || b != bus.Bus(want) {
		t.Fatalf("openDisplay = %v, %v", b, err)
	}

	_, err = s.detector().openDisplay(Display{ID: "msi-3cf0", Bus: "avservice:43"})
	if err == nil || !strings.Contains(err.Error(), "msi-3cf0") {
		t.Fatalf("error = %v, want it to name the display", err)
	}

	for _, bad := range []string{"", "/dev/i2c-13", "avservice:", "avservice:-1", "avservice:x"} {
		if _, err := s.detector().openDisplay(Display{Bus: bad}); err == nil {
			t.Errorf("openDisplay accepted bus %q", bad)
		}
	}
}

func TestDetectGivesSameMonitorsDifferentIDs(t *testing.T) {
	var s services
	s.add(10, msiEDID(t))
	s.add(11, msiEDID(t))

	got, err := s.detector().detect()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"msi-3cf0-0000000000000", "msi-3cf0-0000000000000-2"}
	if len(got) != len(want) {
		t.Fatalf("found %d displays, want %d", len(got), len(want))
	}
	for i, d := range got {
		if d.ID != want[i] {
			t.Errorf("displays[%d].ID = %q, want %q", i, d.ID, want[i])
		}
	}
}

func TestDetectClosesServiceWhenEDIDFails(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*bustest.Fake)
	}{
		{"write fails", func(f *bustest.Fake) { f.FailWrites = 1 }},
		{"read fails", func(f *bustest.Fake) { f.FailReads = 1 }},
		{"edid is not valid", func(f *bustest.Fake) { f.EDID = make([]byte, 128) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s services
			f := s.add(5, msiEDID(t))
			tt.setup(f)

			got, err := s.detector().detect()
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 0 {
				t.Fatalf("got %+v, want nothing", got)
			}
			if !f.Closed {
				t.Fatal("the service was not closed")
			}
		})
	}
}

func TestDetectBusRoundTripsThroughOpenDisplay(t *testing.T) {
	var s services
	want := s.add(18446744073709551615, msiEDID(t))

	found, err := s.detector().detect()
	if err != nil || len(found) != 1 {
		t.Fatalf("detect = %+v, %v", found, err)
	}
	b, err := s.detector().openDisplay(found[0])
	if err != nil || b != bus.Bus(want) {
		t.Fatalf("openDisplay = %v, %v", b, err)
	}
}

func TestOpenDisplayRejectsMalformedBus(t *testing.T) {
	var s services
	s.add(1, msiEDID(t))
	for _, bad := range []string{
		"avservice:18446744073709551616",
		"avservice:+1",
		"avservice: 1",
		"avservice:1 ",
		"avservice:0x1",
		"avservice:1_0",
		"AVSERVICE:1",
		" avservice:1",
	} {
		_, err := s.detector().openDisplay(Display{Bus: bad})
		if err == nil || !strings.Contains(err.Error(), "not a display service") {
			t.Errorf("openDisplay(%q) error = %v, want a bad bus error", bad, err)
		}
	}
}

func TestOpenDisplayErrors(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		openErr error
		wantIs  error
		wantMsg string
	}{
		{"gone", bus.ErrNoAVService, nil, "msi-3cf0 is gone"},
		{"gone wrapped", fmt.Errorf("open: %w", bus.ErrNoAVService), nil, "msi-3cf0 is gone"},
		{"other error passes through", boom, boom, "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := detector{open: func(uint64) (bus.Bus, error) { return nil, tt.openErr }}
			b, err := d.openDisplay(Display{ID: "msi-3cf0", Bus: "avservice:7"})
			if b != nil || err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("openDisplay = %v, %v, want error with %q", b, err, tt.wantMsg)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("error %v does not wrap %v", err, tt.wantIs)
			}
		})
	}
}
