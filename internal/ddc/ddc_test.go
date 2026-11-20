package ddc_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/ddc"
)

type scripted struct {
	writes  [][]byte
	replies [][]byte
}

func (s *scripted) Write(_ uint8, data []byte) error {
	s.writes = append(s.writes, append([]byte(nil), data...))
	return nil
}

func (s *scripted) Read(_ uint8, n int) ([]byte, error) {
	out := make([]byte, n)
	if len(s.replies) > 0 {
		copy(out, s.replies[0])
		s.replies = s.replies[1:]
	}
	return out, nil
}

func (s *scripted) Close() error { return nil }

func newClient(b bus.Bus) (*ddc.Client, *[]time.Duration) {
	var slept []time.Duration
	c := ddc.New(b, ddc.WithSleep(func(d time.Duration) { slept = append(slept, d) }))
	return c, &slept
}

func TestGetVCPWireFormat(t *testing.T) {
	s := &scripted{replies: [][]byte{
		{0x6e, 0x88, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf2},
	}}
	c, slept := newClient(s)

	got, err := c.GetVCP(0x10)
	if err != nil {
		t.Fatalf("GetVCP: %v", err)
	}

	if want := (ddc.Value{Current: 50, Max: 100}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if want := []byte{0x51, 0x82, 0x01, 0x10, 0xac}; !bytes.Equal(s.writes[0], want) {
		t.Fatalf("request = % x, want % x", s.writes[0], want)
	}
	if len(*slept) != 1 || (*slept)[0] < 40*time.Millisecond {
		t.Fatalf("slept %v, want one wait of at least 40ms before reading", *slept)
	}
}

func TestSetVCPWireFormat(t *testing.T) {
	s := &scripted{}
	c, slept := newClient(s)

	if err := c.SetVCP(0x60, 0x0f); err != nil {
		t.Fatalf("SetVCP: %v", err)
	}

	if want := []byte{0x51, 0x84, 0x03, 0x60, 0x00, 0x0f, 0xd7}; !bytes.Equal(s.writes[0], want) {
		t.Fatalf("request = % x, want % x", s.writes[0], want)
	}
	if len(*slept) != 1 || (*slept)[0] < 50*time.Millisecond {
		t.Fatalf("slept %v, want one wait of at least 50ms after writing", *slept)
	}
}

func TestGetAndSetAgainstFakeDisplay(t *testing.T) {
	f := bustest.NewFake()
	f.VCP[0x10] = bustest.VCP{Current: 50, Max: 100}
	c, _ := newClient(f)

	if err := c.SetVCP(0x10, 300); err != nil {
		t.Fatalf("SetVCP: %v", err)
	}
	got, err := c.GetVCP(0x10)
	if err != nil {
		t.Fatalf("GetVCP: %v", err)
	}
	if want := (ddc.Value{Current: 300, Max: 100}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGetVCPUnsupportedFeature(t *testing.T) {
	c, _ := newClient(bustest.NewFake())

	_, err := c.GetVCP(0x10)

	if !errors.Is(err, ddc.ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestGetVCPRetries(t *testing.T) {
	cases := map[string]func(*bustest.Fake){
		"read error":   func(f *bustest.Fake) { f.FailReads = 2 },
		"write error":  func(f *bustest.Fake) { f.FailWrites = 2 },
		"bad checksum": func(f *bustest.Fake) { f.CorruptReplies = 2 },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := bustest.NewFake()
			f.VCP[0x10] = bustest.VCP{Current: 7, Max: 100}
			setup(f)
			c, _ := newClient(f)

			got, err := c.GetVCP(0x10)
			if err != nil {
				t.Fatalf("GetVCP: %v", err)
			}
			if got.Current != 7 {
				t.Fatalf("Current = %d, want 7", got.Current)
			}
		})
	}
}

func TestGetVCPGivesUp(t *testing.T) {
	cases := map[string]struct {
		setup func(*bustest.Fake)
		want  error
	}{
		"display asleep":     {func(f *bustest.Fake) { f.Asleep = true }, ddc.ErrNoReply},
		"always corrupt":     {func(f *bustest.Fake) { f.CorruptReplies = 99 }, ddc.ErrChecksum},
		"bus always failing": {func(f *bustest.Fake) { f.FailReads = 99 }, bustest.ErrInjected},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := bustest.NewFake()
			f.VCP[0x10] = bustest.VCP{Current: 7, Max: 100}
			tc.setup(f)
			c, _ := newClient(f)

			_, err := c.GetVCP(0x10)

			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if got := len(f.Writes); got != 3 {
				t.Fatalf("sent %d requests, want 3", got)
			}
		})
	}
}

func TestGetVCPRejectsReplyForAnotherFeature(t *testing.T) {
	s := &scripted{replies: [][]byte{
		{0x6e, 0x88, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf2},
		{0x6e, 0x88, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf2},
		{0x6e, 0x88, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf2},
	}}
	c, _ := newClient(s)

	_, err := c.GetVCP(0x60)

	if !errors.Is(err, ddc.ErrBadReply) {
		t.Fatalf("err = %v, want ErrBadReply", err)
	}
}

func TestSetVCPReturnsBusError(t *testing.T) {
	f := bustest.NewFake()
	f.FailWrites = 99
	c, _ := newClient(f)

	if err := c.SetVCP(0x10, 1); !errors.Is(err, bustest.ErrInjected) {
		t.Fatalf("err = %v, want injected failure", err)
	}
}

func TestVCPValuesAreBigEndian(t *testing.T) {
	s := &scripted{replies: [][]byte{
		{0x6e, 0x88, 0x02, 0x00, 0x10, 0x00, 0x12, 0x34, 0xab, 0xcd, 0xe4},
	}}
	c, _ := newClient(s)

	got, err := c.GetVCP(0x10)
	if err != nil {
		t.Fatalf("GetVCP: %v", err)
	}
	if want := (ddc.Value{Current: 0xabcd, Max: 0x1234}); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	if err := c.SetVCP(0x10, 0x1234); err != nil {
		t.Fatalf("SetVCP: %v", err)
	}
	if want := []byte{0x51, 0x84, 0x03, 0x10, 0x12, 0x34, 0x8e}; !bytes.Equal(s.writes[1], want) {
		t.Fatalf("request = % x, want % x", s.writes[1], want)
	}
}

func TestGetVCPRejectsMalformedReplies(t *testing.T) {
	cases := map[string]struct {
		reply []byte
		want  error
	}{
		"nothing on the bus":   {nil, ddc.ErrNoReply},
		"wrong source address": {[]byte{0x6f, 0x88, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf3}, ddc.ErrNoReply},
		"length past buffer":   {[]byte{0x6e, 0x8f, 0x02, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf5}, ddc.ErrBadReply},
		"payload too short":    {[]byte{0x6e, 0x83, 0x02, 0x00, 0x10, 0xaf}, ddc.ErrBadReply},
		"wrong opcode":         {[]byte{0x6e, 0x88, 0x03, 0x00, 0x10, 0x00, 0x00, 0x64, 0x00, 0x32, 0xf3}, ddc.ErrBadReply},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s := &scripted{replies: [][]byte{tc.reply, tc.reply, tc.reply}}
			c, _ := newClient(s)

			_, err := c.GetVCP(0x10)

			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(s.writes) != 3 {
				t.Fatalf("sent %d requests, want 3", len(s.writes))
			}
		})
	}
}

func TestGetVCPWaitsBetweenRetries(t *testing.T) {
	f := bustest.NewFake()
	f.VCP[0x10] = bustest.VCP{Current: 7, Max: 100}
	f.FailReads = 2
	c, slept := newClient(f)

	if _, err := c.GetVCP(0x10); err != nil {
		t.Fatalf("GetVCP: %v", err)
	}

	var retryWaits int
	for _, d := range *slept {
		if d >= 100*time.Millisecond {
			retryWaits++
		}
	}
	if retryWaits != 2 {
		t.Fatalf("slept %v, want two waits of at least 100ms between tries", *slept)
	}
}

func TestSetVCPRetries(t *testing.T) {
	f := bustest.NewFake()
	f.VCP[0x10] = bustest.VCP{Current: 1, Max: 100}
	f.FailWrites = 2
	c, _ := newClient(f)

	if err := c.SetVCP(0x10, 42); err != nil {
		t.Fatalf("SetVCP: %v", err)
	}
	if got := f.VCP[0x10].Current; got != 42 {
		t.Fatalf("Current = %d, want 42", got)
	}
}

func TestSetVCPGivesUpAfterThreeTries(t *testing.T) {
	f := bustest.NewFake()
	f.FailWrites = 99
	c, _ := newClient(f)

	_ = c.SetVCP(0x10, 1)

	if tries := 99 - f.FailWrites; tries != 3 {
		t.Fatalf("tried %d writes, want 3", tries)
	}
}

func TestCapabilitiesJoinsFragments(t *testing.T) {
	want := "(prot(monitor)type(lcd)model(MPG 274U E16M)vcp(10 12 60(0F 11 12 1B)))"
	if len(want) <= 64 {
		t.Fatal("test string must span more than two fragments")
	}
	f := bustest.NewFake()
	f.Capabilities = want
	c, _ := newClient(f)

	got, err := c.Capabilities()
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCapabilitiesReadsPastOffset255(t *testing.T) {
	want := "(" + strings.Repeat("vcp(10 12) ", 30) + ")"
	f := bustest.NewFake()
	f.Capabilities = want
	c, _ := newClient(f)

	got, err := c.Capabilities()
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if want := []byte{0x51, 0x83, 0xf3, 0x01, 0x00, 0x4e}; !bytes.Equal(f.Writes[8], want) {
		t.Fatalf("request at offset 256 = % x, want % x", f.Writes[8], want)
	}
}

func TestCapabilitiesRejectsReplyForWrongOffset(t *testing.T) {
	wrong := []byte{0x6e, 0x84, 0xe3, 0x00, 0x20, 'x', 0x00}
	wrong[6] = xorAll(0x50, wrong[:6])
	s := &scripted{replies: [][]byte{wrong, wrong, wrong}}
	c, _ := newClient(s)

	_, err := c.Capabilities()

	if !errors.Is(err, ddc.ErrBadReply) {
		t.Fatalf("err = %v, want ErrBadReply", err)
	}
}

func TestCapabilitiesDisplayAsleep(t *testing.T) {
	f := bustest.NewFake()
	f.Capabilities = "(vcp(10))"
	f.Asleep = true
	c, _ := newClient(f)

	_, err := c.Capabilities()

	if !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("err = %v, want ErrNoReply", err)
	}
}

func xorAll(seed byte, b []byte) byte {
	for _, v := range b {
		seed ^= v
	}
	return seed
}

func TestCapabilitiesDropsControlBytes(t *testing.T) {
	f := bustest.NewFake()
	f.Capabilities = "(vcp\x1b[2J(10))"
	c, _ := newClient(f)

	got, err := c.Capabilities()
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if want := "(vcp[2J(10))"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCapabilitiesStopsOnTinyFragments(t *testing.T) {
	tiny := &endlessCaps{fragment: "x"}
	c, _ := newClient(tiny)

	_, err := c.Capabilities()

	if !errors.Is(err, ddc.ErrBadReply) {
		t.Fatalf("err = %v, want ErrBadReply", err)
	}
	if tiny.requests > 600 {
		t.Fatalf("sent %d requests, want a cap near 512", tiny.requests)
	}
}

func TestCapabilitiesStopsOnEndlessDisplay(t *testing.T) {
	endless := &endlessCaps{fragment: strings.Repeat("x", 32)}
	c, _ := newClient(endless)

	_, err := c.Capabilities()

	if !errors.Is(err, ddc.ErrBadReply) {
		t.Fatalf("err = %v, want ErrBadReply", err)
	}
}

type endlessCaps struct {
	offset   []byte
	fragment string
	requests int
}

func (e *endlessCaps) Write(_ uint8, data []byte) error {
	e.requests++
	e.offset = []byte{data[3], data[4]}
	return nil
}

func (e *endlessCaps) Read(_ uint8, n int) ([]byte, error) {
	payload := append([]byte{0xe3, e.offset[0], e.offset[1]}, []byte(e.fragment)...)
	reply := append([]byte{0x6e, 0x80 | byte(len(payload))}, payload...)
	sum := byte(0x50)
	for _, b := range reply {
		sum ^= b
	}
	out := make([]byte, n)
	copy(out, append(reply, sum))
	return out, nil
}

func (e *endlessCaps) Close() error { return nil }
