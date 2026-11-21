package osd

import "testing"

func TestTheSystemIndicatorIsFound(t *testing.T) {
	if !Available() {
		t.Skip("this macOS version has no indicator monctl knows")
	}
}
