package ui

import (
	"testing"
	"time"
)

func TestFormatSize_RendersReadableUnits(t *testing.T) {
	cases := map[int64]string{
		0:              "0 B",
		512:            "512 B",
		1024:           "1.0 KiB",
		1536:           "1.5 KiB",
		1048576:        "1.0 MiB",
		5 * 1073741824: "5.0 GiB",
		-1:             "?",
	}
	for input, want := range cases {
		if got := formatSize(input); got != want {
			t.Errorf("formatSize(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatModified_EmptyWhenUnknown(t *testing.T) {
	if got := formatModified(time.Time{}); got != "" {
		t.Errorf("formatModified(zero) = %q, want an empty string", got)
	}
	stamp := time.Date(2026, 11, 12, 9, 30, 0, 0, time.UTC)
	if got := formatModified(stamp); got != "2026-11-12 09:30" {
		t.Errorf("formatModified = %q, want the short date form", got)
	}
}
