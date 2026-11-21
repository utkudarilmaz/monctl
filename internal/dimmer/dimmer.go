// Package dimmer changes the brightness for a program that stays running.
// It remembers the brightness, so a key press costs one write, and it merges
// presses that arrive while a write is under way.
package dimmer

import (
	"sync"
	"sync/atomic"
	"time"
)

// stale is how long a remembered brightness is trusted. The monitor's own
// buttons or another computer can change it in the meantime.
const stale = 5 * time.Second

// Monitor reads and writes the raw brightness of one monitor.
type Monitor interface {
	Level() (current, limit int, err error)
	Write(value int) error
}

// Dimmer applies brightness changes to a Monitor.
type Dimmer struct {
	monitor Monitor
	report  func(error)
	now     func() time.Time

	mu      sync.Mutex
	known   bool
	seen    time.Time
	target  int
	limit   int
	written int
	writing bool
	idle    sync.WaitGroup

	usable atomic.Bool
}

// New returns a Dimmer. Write errors go to report.
func New(monitor Monitor, report func(error)) *Dimmer {
	return &Dimmer{monitor: monitor, report: report, now: time.Now}
}

// Usable reports whether the last contact with the monitor worked. It never
// blocks.
func (d *Dimmer) Usable() bool { return d.usable.Load() }

// Probe reads the monitor and updates Usable.
func (d *Dimmer) Probe() {
	d.mu.Lock()
	defer d.mu.Unlock()
	_ = d.refresh()
}

// Step changes the brightness by delta raw units and returns the new target
// and the monitor's maximum.
func (d *Dimmer) Step(delta int) (int, int, error) {
	return d.change(func(target, _ int) int { return target + delta })
}

// StepPercent changes the brightness by percent of the monitor's maximum.
func (d *Dimmer) StepPercent(percent int) (int, int, error) {
	return d.change(func(target, limit int) int {
		delta := percent * limit / 100
		if delta == 0 && percent != 0 {
			delta = percent / max(percent, -percent)
		}
		return target + delta
	})
}

// Set changes the brightness to value raw units.
func (d *Dimmer) Set(value int) (int, int, error) {
	return d.change(func(int, int) int { return value })
}

// Wait returns when no write is under way.
func (d *Dimmer) Wait() { d.idle.Wait() }

func (d *Dimmer) change(next func(target, limit int) int) (int, int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.known || (!d.writing && d.now().Sub(d.seen) > stale) {
		if err := d.refresh(); err != nil {
			return 0, 0, err
		}
	}
	d.target = max(0, min(next(d.target, d.limit), d.limit))
	if !d.writing && d.target != d.written {
		d.writing = true
		d.idle.Add(1)
		go d.flush()
	}
	return d.target, d.limit, nil
}

func (d *Dimmer) refresh() error {
	current, limit, err := d.monitor.Level()
	d.usable.Store(err == nil)
	if err != nil {
		d.known = false
		return err
	}
	d.known, d.seen = true, d.now()
	d.target, d.written, d.limit = current, current, limit
	return nil
}

func (d *Dimmer) flush() {
	defer d.idle.Done()
	for {
		d.mu.Lock()
		target := d.target
		if target == d.written {
			d.writing = false
			d.mu.Unlock()
			return
		}
		d.mu.Unlock()

		err := d.monitor.Write(target)

		d.mu.Lock()
		d.usable.Store(err == nil)
		if err != nil {
			d.known, d.writing = false, false
			d.mu.Unlock()
			d.report(err)
			return
		}
		d.written, d.seen = target, d.now()
		d.mu.Unlock()
	}
}
