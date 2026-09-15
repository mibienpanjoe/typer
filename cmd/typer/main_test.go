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

func TestFireRequiresID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"fire"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d stderr=%s", code, stderr.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"wat"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), `commande inconnue "wat"`) {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestAtFlagValueIsNotACommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	_ = run([]string{"--at", "06:34", "-m", "continue", "--yes"}, &stdout, &stderr)
	if strings.Contains(stderr.String(), "commande inconnue") {
		t.Fatalf("06:34 was parsed as a command: %s", stderr.String())
	}
}
