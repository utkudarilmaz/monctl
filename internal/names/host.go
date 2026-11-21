package names

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Host returns the name of this computer: the name the user gave it on macOS,
// and the host name without its domain elsewhere.
func Host() string {
	if runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, "/usr/sbin/scutil", "--get", "ComputerName").Output(); err == nil {
			if name := Clean(string(out)); name != "" {
				return name
			}
		}
	}
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	short, _, _ := strings.Cut(host, ".")
	return Clean(short)
}
