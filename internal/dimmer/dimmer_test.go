package dimmer

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeMonitor struct {
	mu       sync.Mutex
	current  int
	limit    int
	reads    int
	writes   []int
	levelErr error
	writeErr error
	gate     chan struct{}
}

func (f *fakeMonitor) Level() (int, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return f.current, f.limit, f.levelErr
}

func (f *fakeMonitor) Write(value int) error {
	if f.gate != nil {
		<-f.gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.writes = append(f.writes, value)
	f.current = value
	return nil
}

func newDimmer(f *fakeMonitor) (*Dimmer, *[]error) {
	var reported []error
	d := New(f, func(err error) { reported = append(reported, err) })
	return d, &reported
}

func TestStepReadsOnceThenOnlyWrites(t *testing.T) {
	f := &fakeMonitor{current: 50, limit: 100}
	d, _ := newDimmer(f)

	for _, want := range []int{55, 60, 65} {
		got, limit, err := d.Step(5)
		d.Wait()
		if err != nil || got != want || limit != 100 {
			t.Fatalf("Step = %d, %d, %v; want %d, 100", got, limit, err, want)
		}
	}
	if f.reads != 1 || len(f.writes) != 3 || f.current != 65 {
		t.Fatalf("reads %d, writes %v, current %d", f.reads, f.writes, f.current)
	}
}

func TestStepsDuringAWriteAreMerged(t *testing.T) {
	f := &fakeMonitor{current: 50, limit: 100, gate: make(chan struct{})}
	d, _ := newDimmer(f)

	for range 6 {
		if _, _, err := d.Step(5); err != nil {
			t.Fatal(err)
		}
	}
	close(f.gate)
	d.Wait()

	if f.current != 80 {
		t.Fatalf("monitor ended at %d, want 80", f.current)
	}
	if len(f.writes) > 2 {
		t.Fatalf("writes %v, want the six steps merged into at most two writes", f.writes)
	}
}

func TestStepStaysInsideTheRange(t *testing.T) {
	f := &fakeMonitor{current: 95, limit: 100}
	d, _ := newDimmer(f)

	if got, _, _ := d.Step(50); got != 100 {
		t.Fatalf("Step up = %d, want 100", got)
	}
	d.Wait()
	if got, _, _ := d.Step(-500); got != 0 {
		t.Fatalf("Step down = %d, want 0", got)
	}
	d.Wait()
	before := len(f.writes)
	if got, _, _ := d.Step(-5); got != 0 || func() bool { d.Wait(); return len(f.writes) != before }() {
		t.Fatal("a step at the edge must not write again")
	}
}

func TestStepPercentUsesTheMonitorMaximum(t *testing.T) {
	tests := []struct{ limit, start, percent, want int }{
		{100, 50, 5, 55}, {255, 100, 10, 125}, {50, 10, -10, 5}, {10, 5, 5, 6}, {10, 5, -5, 4},
	}
	for _, tt := range tests {
		f := &fakeMonitor{current: tt.start, limit: tt.limit}
		d, _ := newDimmer(f)
		got, _, err := d.StepPercent(tt.percent)
		d.Wait()
		if err != nil || got != tt.want {
			t.Errorf("StepPercent(%d) from %d of %d = %d, %v; want %d", tt.percent, tt.start, tt.limit, got, err, tt.want)
		}
	}
}

func TestSet(t *testing.T) {
	f := &fakeMonitor{current: 50, limit: 100}
	d, _ := newDimmer(f)

	got, _, err := d.Set(250)
	d.Wait()
	if err != nil || got != 100 || f.current != 100 {
		t.Fatalf("Set(250) = %d, %v, monitor at %d", got, err, f.current)
	}
}

func TestTheRememberedValueGoesStale(t *testing.T) {
	f := &fakeMonitor{current: 50, limit: 100}
	d, _ := newDimmer(f)
	clock := time.Unix(0, 0)
	d.now = func() time.Time { return clock }

	_, _, _ = d.Step(5)
	d.Wait()
	f.current = 20
	clock = clock.Add(stale + time.Second)

	got, _, _ := d.Step(5)
	d.Wait()
	if got != 25 || f.reads != 2 {
		t.Fatalf("Step after a pause = %d with %d reads, want 25 and 2", got, f.reads)
	}
}

func TestErrorsAndUsable(t *testing.T) {
	f := &fakeMonitor{current: 50, limit: 100, levelErr: errors.New("silent")}
	d, reported := newDimmer(f)

	if d.Usable() {
		t.Fatal("a new dimmer is not usable before it has seen the monitor")
	}
	if _, _, err := d.Step(5); err == nil || d.Usable() {
		t.Fatal("a failed read must fail the step")
	}

	f.levelErr = nil
	d.Probe()
	if !d.Usable() {
		t.Fatal("Probe must mark a working monitor usable")
	}

	f.writeErr = errors.New("write failed")
	_, _, _ = d.Step(5)
	d.Wait()
	if d.Usable() || len(*reported) != 1 {
		t.Fatalf("usable %v, reported %v; want a reported write error", d.Usable(), *reported)
	}

	f.writeErr = nil
	if got, _, err := d.Step(5); err != nil || got != 55 {
		t.Fatalf("Step after a write error = %d, %v; want a fresh read and 55", got, err)
	}
	d.Wait()
	if !d.Usable() {
		t.Fatal("a working write must mark the monitor usable again")
	}
}
