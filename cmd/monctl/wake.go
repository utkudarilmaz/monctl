package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/utkudarilmaz/monctl/internal/monitor"
	"github.com/utkudarilmaz/monctl/internal/names"
)

const (
	beforeTimeout = 20 * time.Second
	beforeGrace   = time.Second
	outputKept    = 4096
	messageRunes  = 200
)

const (
	switchAttempts   = 3
	switchChecks     = 7
	switchCheckEvery = 2 * time.Second
)

var (
	errBefore   = errors.New("the command to run before switching failed, switching anyway")
	errCameBack = errors.New("the monitor came back to this computer, the other one gave no picture")
)

func (a app) wakeScreen(opts options) int {
	if len(opts.args) > 0 || opts.json || opts.raw || opts.display != "" {
		return a.usageError("wake takes no value and no flags")
	}
	if a.state != "" {
		if name := a.host(); name != "" {
			_, _ = fmt.Fprintln(a.stdout, names.Line(name))
		}
	}
	if err := a.wake(); err != nil {
		return a.fail(err)
	}
	return 0
}

// switchTo runs the input's before command, switches, and checks that the
// monitor stays away. A monitor that finds no picture on the new input comes
// back and answers this computer again. A failed switch can still wake the
// other computer through the cable, so it is tried more than once.
func (a app) switchTo(s *session, name string) (monitor.Input, error) {
	in, err := s.monitor.Lookup(name)
	if err != nil {
		return monitor.Input{}, err
	}
	origin, err := a.source(s)
	if err != nil || (origin.Known && origin.Input.ID == in.ID) {
		return s.monitor.SetInput(in.ID)
	}
	if before := s.config.Inputs[in.ID].Before; len(before) > 0 {
		output, err := a.exec(before)
		if err != nil {
			_, _ = fmt.Fprintf(a.stderr, "monctl: %v: %v\n", errBefore, err)
		}
		if a.remember(s, in.ID, names.Parse(output)) {
			in, _ = s.monitor.Lookup(in.ID)
		}
	}
	for range switchAttempts {
		if _, err := s.monitor.SetInput(in.ID); err != nil {
			return in, err
		}
		if !a.cameBack(s, origin.Value) {
			return in, nil
		}
	}
	return in, errCameBack
}

// cameBack watches the monitor after a switch and reports whether it shows
// this computer again.
func (a app) cameBack(s *session, origin uint16) bool {
	for range switchChecks {
		a.pause(switchCheckEvery)
		if src, err := s.monitor.Source(); err == nil && src.Value == origin {
			return true
		}
	}
	return false
}

func runCommand(argv []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var output tail
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = &output, &output
	// Without this a child of the killed program that still holds the output
	// keeps Run waiting past the timeout.
	cmd.WaitDelay = beforeGrace
	err := cmd.Run()
	text := string(output)
	switch line := lastLine(text); {
	case err == nil:
		return text, nil
	case ctx.Err() != nil:
		return text, fmt.Errorf("%s did not finish in %s", argv[0], timeout)
	case line != "":
		return text, fmt.Errorf("%s: %s", argv[0], line)
	default:
		return text, fmt.Errorf("%s: %w", argv[0], err)
	}
}

// tail keeps only the end of what is written to it.
type tail []byte

func (t *tail) Write(p []byte) (int, error) {
	*t = append(*t, p...)
	if extra := len(*t) - outputKept; extra > 0 {
		*t = (*t)[extra:]
	}
	return len(p), nil
}

// lastLine returns the last line of text in a form that is safe to print: the
// text may come from another computer.
func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	line := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, lines[len(lines)-1])))
	return string(line[:min(len(line), messageRunes)])
}
