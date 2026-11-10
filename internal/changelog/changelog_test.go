package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validFragment = `---
timestamp: 2026-07-15T20:45:00Z
agent: claude-code
files:
  - internal/connection/info.go
  - cmd/mremoteng/main.go
---

Add the base connection model.
`

func TestParseFragment_Valid_ReturnsCompleteEntry(t *testing.T) {
	e, err := ParseFragment([]byte(validFragment))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 7, 15, 20, 45, 0, 0, time.UTC)
	if !e.Timestamp.Equal(want) {
		t.Errorf("timestamp = %v, want %v", e.Timestamp, want)
	}
	if e.Agent != "claude-code" {
		t.Errorf("agent = %q, want claude-code", e.Agent)
	}
	if e.Summary != "Add the base connection model." {
		t.Errorf("summary = %q", e.Summary)
	}
	if len(e.Files) != 2 || e.Files[0] != "internal/connection/info.go" || e.Files[1] != "cmd/mremoteng/main.go" {
		t.Errorf("files = %v", e.Files)
	}
}

func TestParseFragment_Invalid_ReturnsError(t *testing.T) {
	cases := map[string]string{
		"no front matter":   "Add something.\n",
		"no closing marker": "---\ntimestamp: 2026-07-15T20:45:00Z\nagent: x\nAdd something.\n",
		"no timestamp":      "---\nagent: x\n---\nAdd something.\n",
		"no agent":          "---\ntimestamp: 2026-07-15T20:45:00Z\n---\nAdd something.\n",
		"no summary":        "---\ntimestamp: 2026-07-15T20:45:00Z\nagent: x\n---\n\n",
		"broken timestamp":  "---\ntimestamp: yesterday\nagent: x\n---\nAdd something.\n",
	}
	for name, input := range cases {
		if _, err := ParseFragment([]byte(input)); err == nil {
			t.Errorf("%s: expected an error, got none", name)
		}
	}
}

func TestRender_ChronologicalOrder(t *testing.T) {
	entries := []Entry{
		{Timestamp: time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC), Agent: "opencode", Summary: "Change C."},
		{Timestamp: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Agent: "claude-code", Summary: "Change A.", Files: []string{"a.go"}},
		{Timestamp: time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC), Agent: "opencode", Summary: "Change B."},
	}
	out := Render(entries)

	// The most recent date section comes first.
	pos16 := strings.Index(out, "## 2026-07-16")
	pos15 := strings.Index(out, "## 2026-07-15")
	if pos16 == -1 || pos15 == -1 || pos16 > pos15 {
		t.Fatalf("wrong date order:\n%s", out)
	}
	// Within the same date, ascending time order.
	posA := strings.Index(out, "Change A.")
	posB := strings.Index(out, "Change B.")
	if posA == -1 || posB == -1 || posA > posB {
		t.Errorf("wrong intra-day order:\n%s", out)
	}
	if !strings.Contains(out, "  - `a.go`\n") {
		t.Errorf("affected files missing:\n%s", out)
	}
	// The explanation comes first and the metadata line (date/time + agent)
	// closes the entry, after the file list.
	entryA := "- Change A.\n  - `a.go`\n  - _2026-07-15 08:00:00 UTC — claude-code_\n"
	if !strings.Contains(out, entryA) {
		t.Errorf("entry layout is wrong, want explanation first and metadata last:\n%s", out)
	}
	if !strings.HasPrefix(out, Header) {
		t.Errorf("generated header missing")
	}
}

func TestParseFragment_DescriptionSeparatedByBlankLine(t *testing.T) {
	fragment := `---
timestamp: 2026-07-22T20:00:00Z
agent: opencode
files:
  - main.go
---

Fix the parser

The body parser split on the wrong delimiter. This caused descriptions
to be lost. Added a test and fixed the split logic.
`
	e, err := ParseFragment([]byte(fragment))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Summary != "Fix the parser" {
		t.Errorf("summary = %q, want %q", e.Summary, "Fix the parser")
	}
	wantDesc := "The body parser split on the wrong delimiter. This caused descriptions\nto be lost. Added a test and fixed the split logic."
	if e.Description != wantDesc {
		t.Errorf("description = %q, want %q", e.Description, wantDesc)
	}
}

func TestParseFragment_NoDescription_OnlySummary(t *testing.T) {
	fragment := `---
timestamp: 2026-07-22T20:00:00Z
agent: opencode
---

Just a summary without details.
`
	e, err := ParseFragment([]byte(fragment))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Summary != "Just a summary without details." {
		t.Errorf("summary = %q", e.Summary)
	}
	if e.Description != "" {
		t.Errorf("description = %q, want empty", e.Description)
	}
}

func TestRender_DescriptionIndentedUnderSummary(t *testing.T) {
	entries := []Entry{
		{
			Timestamp:   time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC),
			Agent:       "opencode",
			Summary:     "Add feature X.",
			Description: "Feature X does two things:\nline one and line two.",
			Files:       []string{"a.go"},
		},
	}
	out := Render(entries)
	if !strings.Contains(out, "- Add feature X.\n") {
		t.Errorf("summary line missing:\n%s", out)
	}
	if !strings.Contains(out, "\n  Feature X does two things:\n  line one and line two.\n") {
		t.Errorf("description not indented under summary:\n%s", out)
	}
	if !strings.Contains(out, "  - `a.go`\n") {
		t.Errorf("file list missing:\n%s", out)
	}
}

func TestRender_NoDescription_KeepsOldLayout(t *testing.T) {
	entries := []Entry{
		{
			Timestamp: time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC),
			Agent:     "opencode",
			Summary:   "Simple change.",
			Files:     []string{"b.go"},
		},
	}
	out := Render(entries)
	expected := "- Simple change.\n  - `b.go`\n  - _2026-07-22 10:00:00 UTC — opencode_\n"
	if !strings.Contains(out, expected) {
		t.Errorf("layout without description is wrong, want:\n%s\ngot:\n%s", expected, out)
	}
}

func TestLoadDir_IgnoresReadmeAndSorts(t *testing.T) {
	dir := t.TempDir()
	older := "---\ntimestamp: 2026-07-14T10:00:00Z\nagent: opencode\n---\nOld change.\n"
	newer := "---\ntimestamp: 2026-07-15T10:00:00Z\nagent: claude-code\n---\nNew change.\n"
	writeFile(t, filepath.Join(dir, "zzz-old.md"), older)
	writeFile(t, filepath.Join(dir, "aaa-new.md"), newer)
	writeFile(t, filepath.Join(dir, "README.md"), "this is not a fragment")

	entries, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2 (README must be ignored)", len(entries))
	}
	if entries[0].Summary != "Old change." || entries[1].Summary != "New change." {
		t.Errorf("wrong chronological order: %+v", entries)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
