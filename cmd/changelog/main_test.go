package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSplitFiles_BlanksAndSpaces_AreDropped(t *testing.T) {
	got := splitFiles(" a.go, ,b.go ,")
	want := []string{"a.go", "b.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitFiles = %v, want %v", got, want)
	}
}

func TestRenderFragment_WithDescriptionAndFiles_HasAllSections(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	got := renderFragment(now, "claude-code", " Fix it ", "Why.", []string{"a.go"})
	want := "---\ntimestamp: 2026-01-02T03:04:05Z\nagent: claude-code\nfiles:\n  - a.go\n---\n\nFix it\n\nWhy.\n"
	if got != want {
		t.Fatalf("renderFragment =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderFragment_NoDescriptionNoFiles_OmitsThem(t *testing.T) {
	got := renderFragment(time.Unix(0, 0).UTC(), "a", "s", "  ", nil)
	if strings.Contains(got, "files:") || strings.HasSuffix(got, "\n\n") {
		t.Fatalf("renderFragment = %q, want no files block and no trailing blank section", got)
	}
}
