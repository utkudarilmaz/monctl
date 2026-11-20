// Command monctl changes external monitor settings over DDC/CI.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/utkudarilmaz/monctl/internal/display"
)

var version = "dev"

const usage = `Usage: monctl <command>

Commands:
  list      Show the external monitors monctl can reach
  version   Print the monctl version
`

const accessHelp = `monctl: not allowed to open the monitor control channel.

On Linux, install the udev rule that gives your user access:

  sudo cp packaging/linux/60-monctl.rules /etc/udev/rules.d/
  sudo udevadm control -R
  sudo udevadm trigger -s i2c-dev

Then run monctl again.
`

type app struct {
	stdout io.Writer
	stderr io.Writer
	detect func() ([]display.Display, error)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return app{stdout: stdout, stderr: stderr, detect: display.Detect}.run(args)
}

func (a app) run(args []string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(a.stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		_, _ = fmt.Fprintf(a.stdout, "monctl %s\n", version)
		return 0
	case "list":
		return a.list()
	default:
		_, _ = fmt.Fprintf(a.stderr, "monctl: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func (a app) list() int {
	displays, err := a.detect()
	if errors.Is(err, display.ErrAccess) {
		_, _ = fmt.Fprint(a.stderr, accessHelp)
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "monctl: %v\n", err)
		return 1
	}
	if len(displays) == 0 {
		_, _ = fmt.Fprintln(a.stdout, "No external monitor found.")
		return 0
	}

	w := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tCONNECTOR\tBUS")
	for _, d := range displays {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Info.Name, d.Connector, d.Bus)
	}
	_ = w.Flush()
	return 0
}
