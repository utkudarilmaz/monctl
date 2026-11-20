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
)

const (
	beforeTimeout = 20 * time.Second
	beforeGrace   = time.Second
	outputKept    = 4096
	messageRunes  = 200
)

var errBefore = errors.New("the command to run before switching failed, so the picture was not moved")

func (a app) wakeScreen(opts options) int {
	if len(opts.args) > 0 || opts.json || opts.raw || opts.display != "" {
		return a.usageError("wake takes no value and no flags")
	}
	if err := a.wake(); err != nil {
		return a.fail(err)
	}
	return 0
}

// switchTo runs the input's before command, then switches. A monitor sent to
// a computer whose screen is asleep finds no picture, so a failed command
// stops the switch.
func (a app) switchTo(s *session, name string) (monitor.Input, error) {
	in, err := s.monitor.Lookup(name)
	if err != nil {
		return monitor.Input{}, err
	}
	if before := s.config.Inputs[in.ID].Before; len(before) > 0 {
		if err := a.exec(before); err != nil {
			return in, fmt.Errorf("%w: %w", errBefore, err)
		}
	}
	return s.monitor.SetInput(in.ID)
}

func runCommand(argv []string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var output tail
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = &output, &output
	// Without this a child of the killed program that still holds the output
	// keeps Run waiting past the timeout.
	cmd.WaitDelay = beforeGrace
	err := cmd.Run()
	switch line := lastLine(string(output)); {
	case err == nil:
		return nil
	case ctx.Err() != nil:
		return fmt.Errorf("%s did not finish in %s", argv[0], timeout)
	case line != "":
		return fmt.Errorf("%s: %s", argv[0], line)
	default:
		return fmt.Errorf("%s: %w", argv[0], err)
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
