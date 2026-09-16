package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParseClockUsesTodayWhenStillInTheFuture(t *testing.T) {
	now := time.Date(2026, 9, 15, 3, 30, 0, 0, time.Local)
	got, err := ParseClock("06:34", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseClockUsesTomorrowWhenAlreadyPastToday(t *testing.T) {
	now := time.Date(2026, 9, 15, 7, 0, 0, 0, time.Local)
	got, err := ParseClock("06:34", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 16, 6, 34, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseClockUsesTomorrowWhenEqualToNow(t *testing.T) {
	now := time.Date(2026, 9, 15, 6, 34, 0, 0, time.Local)
	got, err := ParseClock("06:34", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 16, 6, 34, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseClockRejectsInvalid(t *testing.T) {
	now := time.Date(2026, 9, 15, 3, 0, 0, 0, time.Local)
	for _, in := range []string{"", "6h34", "25:00", "12:60", "ab:cd", "6:34:01", "6:3", "06:34junk", " 06:34"} {
		if _, err := ParseClock(in, now); err == nil {
			t.Errorf("%q: want error", in)
		}
	}
}

func TestValidateMessageTrimsAndKeepsText(t *testing.T) {
	got, err := ValidateMessage("  continue the task  ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "continue the task" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateMessageRejectsEmpty(t *testing.T) {
	if _, err := ValidateMessage("   "); err == nil {
		t.Fatal("want error")
	}
}

func TestValidateMessageRejectsTooLong(t *testing.T) {
	if _, err := ValidateMessage(strings.Repeat("a", MaxMessageBytes+1)); err == nil {
		t.Fatal("want error")
	}
}

func TestValidateMessageAcceptsMaxLength(t *testing.T) {
	in := strings.Repeat("a", MaxMessageBytes)
	got, err := ValidateMessage(in)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatal("truncated")
	}
}

func TestNormalizeSubmitKeyDefaultsToEnter(t *testing.T) {
	got, err := NormalizeSubmitKey("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "enter" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeSubmitKeyAcceptsEnterAliases(t *testing.T) {
	for _, in := range []string{"enter", "Return", " return "} {
		got, err := NormalizeSubmitKey(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != "enter" {
			t.Fatalf("%q: got %q", in, got)
		}
	}
}

func TestNormalizeSubmitKeyAcceptsCtrlJAliases(t *testing.T) {
	for _, in := range []string{"ctrl+j", "Control+J", "c-j", "cj"} {
		got, err := NormalizeSubmitKey(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != "ctrl+j" {
			t.Fatalf("%q: got %q", in, got)
		}
	}
}

func TestNormalizeSubmitKeyRejectsUnknown(t *testing.T) {
	if _, err := NormalizeSubmitKey("shift+enter"); err == nil {
		t.Fatal("want error")
	}
}
