// Package profiles holds the monitor profiles that ship with monctl.
package profiles

import "embed"

// FS contains one JSON profile per supported monitor.
//
//go:embed *.json
var FS embed.FS
