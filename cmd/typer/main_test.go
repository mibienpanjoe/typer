package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpMentionsCommandsAndFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	out := stdout.String() + stderr.String()
	for _, want := range []string{"list", "cancel", "fire", "--at", "-m"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q\n%s", want, out)
		}
	}
}
