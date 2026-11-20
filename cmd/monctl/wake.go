package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/utkudarilmaz/monctl/internal/monitor"
)

const beforeTimeout = 20 * time.Second

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

func runCommand(argv []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), beforeTimeout)
	defer cancel()

	var output bytes.Buffer
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	if ctx.Err() != nil {
		return fmt.Errorf("%s did not finish in %s", argv[0], beforeTimeout)
	}
	if err != nil {
		if line := lastLine(output.String()); line != "" {
			return fmt.Errorf("%s: %s", argv[0], line)
		}
		return fmt.Errorf("%s: %w", argv[0], err)
	}
	return nil
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
