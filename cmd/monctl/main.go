// Command monctl changes external monitor settings over DDC/CI.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/display"
	"github.com/utkudarilmaz/monctl/internal/monitor"
	"github.com/utkudarilmaz/monctl/internal/profile"
	"github.com/utkudarilmaz/monctl/internal/tray"
	"github.com/utkudarilmaz/monctl/profiles"
)

var version = "dev"

const usage = `Usage: monctl <command> [flags] [value]

Commands:
  list                 Show the external monitors monctl can reach
  brightness           Print the brightness
  brightness <n>       Set the brightness to n
  brightness +<n>      Raise the brightness by n
  brightness -<n>      Lower the brightness by n
  input                Print the current input and the inputs you can switch to
  input <name>         Switch the monitor to another input
  input -raw <value>   Switch with a raw value, for monitors without a profile
  probe                Print what a monitor reports, to help add a new model
  tray                 Show a menu in the system tray until you quit it
  version              Print the monctl version

Flags:
  -d <id>   Pick a monitor by its id from "monctl list"
  -json     Print JSON
`

const accessHelp = `monctl: not allowed to open the monitor control channel.

On Linux, install the udev rule that gives your user access:

  sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
  sudo udevadm control -R
  sudo udevadm trigger -s i2c-dev

Then run monctl again.
`

const silentHelp = `monctl: the monitor did not answer.

A monitor only answers on the input it is showing. It may be showing another
computer, or this computer's screen may be asleep.
`

type app struct {
	stdout   io.Writer
	stderr   io.Writer
	detect   func() ([]display.Display, error)
	open     func(display.Display) (bus.Bus, error)
	profiles func() ([]profile.Profile, error)
	config   func() (config.Config, error)
	ddc      []ddc.Option
	runTray  func(tray.Controller, func(error)) error
}

var errNoMonitor = errors.New("no external monitor found")

type options struct {
	display string
	json    bool
	raw     bool
	args    []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return app{
		stdout:   stdout,
		stderr:   stderr,
		detect:   display.Detect,
		open:     display.Open,
		profiles: loadProfiles,
		config:   loadConfig,
		runTray:  tray.Run,
	}.run(args)
}

func loadProfiles() ([]profile.Profile, error) {
	dir, err := config.Dir()
	if err != nil {
		return profile.Load(profiles.FS)
	}
	return profile.Load(os.DirFS(filepath.Join(dir, "profiles")), profiles.FS)
}

func loadConfig() (config.Config, error) {
	dir, err := config.Dir()
	if err != nil {
		return config.Config{}, nil
	}
	return config.Load(dir)
}

func (a app) run(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(a.stderr, usage)
		return 2
	}

	commands := map[string]func(options) int{
		"list":       a.list,
		"brightness": a.brightness,
		"input":      a.input,
		"probe":      a.probe,
		"tray":       a.tray,
	}

	switch command, ok := commands[args[0]]; {
	case args[0] == "version":
		_, _ = fmt.Fprintf(a.stdout, "monctl %s\n", version)
		return 0
	case ok:
		opts, err := parseOptions(args[0], args[1:])
		if err != nil {
			_, _ = fmt.Fprintf(a.stderr, "monctl: %v\n\n%s", err, usage)
			return 2
		}
		return command(opts)
	default:
		_, _ = fmt.Fprintf(a.stderr, "monctl: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

var signedNumber = regexp.MustCompile(`^[+-][0-9]+$`)

// parseOptions lets flags and values come in any order, and keeps values
// such as -10 away from the flag parser.
func parseOptions(command string, args []string) (options, error) {
	var opts options
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.display, "d", "", "")
	fs.BoolVar(&opts.json, "json", false, "")
	fs.BoolVar(&opts.raw, "raw", false, "")

	var flags []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-d" && i+1 < len(args):
			flags = append(flags, arg, args[i+1])
			i++
		case len(arg) > 1 && arg[0] == '-' && !signedNumber.MatchString(arg):
			flags = append(flags, arg)
		default:
			opts.args = append(opts.args, arg)
		}
	}
	if err := fs.Parse(flags); err != nil {
		return options{}, err
	}
	return opts, nil
}

func (a app) fail(err error) int {
	switch {
	case errors.Is(err, display.ErrAccess):
		_, _ = fmt.Fprint(a.stderr, accessHelp)
	case errors.Is(err, ddc.ErrNoReply):
		_, _ = fmt.Fprint(a.stderr, silentHelp)
	default:
		_, _ = fmt.Fprintf(a.stderr, "monctl: %v\n", err)
	}
	return 1
}

func (a app) usageError(format string, args ...any) int {
	_, _ = fmt.Fprintf(a.stderr, "monctl: "+format+"\n", args...)
	return 2
}

func (a app) printJSON(v any) int {
	enc := json.NewEncoder(a.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return a.fail(err)
	}
	return 0
}

func (a app) pick(id string) (display.Display, error) {
	displays, err := a.detect()
	if err != nil {
		return display.Display{}, err
	}
	if id != "" {
		for _, d := range displays {
			if d.ID == id {
				return d, nil
			}
		}
		return display.Display{}, fmt.Errorf("no monitor with id %q, see \"monctl list\"", id)
	}
	switch len(displays) {
	case 0:
		return display.Display{}, errNoMonitor
	case 1:
		return displays[0], nil
	default:
		return display.Display{}, errors.New("more than one monitor found, pick one with -d <id> from \"monctl list\"")
	}
}

type session struct {
	display display.Display
	profile *profile.Profile
	client  *ddc.Client
	monitor *monitor.Monitor
	close   func()
}

func (a app) connect(id string) (*session, error) {
	disp, err := a.pick(id)
	if err != nil {
		return nil, err
	}
	all, err := a.profiles()
	if err != nil {
		return nil, err
	}
	cfg, err := a.config()
	if err != nil {
		return nil, err
	}
	b, err := a.open(disp)
	if err != nil {
		return nil, err
	}

	s := &session{display: disp, client: ddc.New(b, a.ddc...), close: func() { _ = b.Close() }}
	if p, ok := profile.Find(all, disp.Info); ok {
		s.profile = &p
	}
	s.monitor = monitor.New(s.client, s.profile, cfg)
	return s, nil
}
