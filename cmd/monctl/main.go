// Command monctl changes external monitor settings over DDC/CI.
package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

const usage = `Usage: monctl <command>

Commands:
  version   Print the monctl version
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "version":
		_, _ = fmt.Fprintf(stdout, "monctl %s\n", version)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "monctl: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}
