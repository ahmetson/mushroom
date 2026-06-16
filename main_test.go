package main

import (
	"bytes"
	"testing"
)

func TestRunWritesProjectName(t *testing.T) {
	var stdout bytes.Buffer

	if err := run(&stdout); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	const want = "mushroom\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}
