package main

import (
	"bytes"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"version"}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := out.String(), "monctl dev\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"nope"}, &out, &errOut)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if errOut.Len() == 0 {
		t.Fatal("expected a message on stderr")
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer

	if code := run(nil, &out, &errOut); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}
