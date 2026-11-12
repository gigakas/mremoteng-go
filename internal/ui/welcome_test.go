package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

// TestWelcomeHelp_DocumentsEveryIconButton guards the start page reference: every
// icon-only button in the window must have a line, because an icon nobody hovers
// is an icon nobody understands.
func TestWelcomeHelp_DocumentsEveryIconButton(t *testing.T) {
	// The window has eleven label-less buttons; the text buttons are documented
	// too, so the list must be at least that long.
	if len(welcomeHelp) < 11 {
		t.Fatalf("welcomeHelp has %d rows, want at least one per icon-only button", len(welcomeHelp))
	}
	seen := map[string]bool{}
	for _, row := range welcomeHelp {
		if strings.TrimSpace(row.name) == "" || strings.TrimSpace(row.what) == "" {
			t.Errorf("row %+v is incomplete", row)
		}
		if strings.EqualFold(strings.TrimSpace(row.name), strings.TrimSpace(row.what)) {
			t.Errorf("row %q does not explain anything", row.name)
		}
		if !strings.HasSuffix(row.what, ".") {
			t.Errorf("row %q does not end its sentence: %q", row.name, row.what)
		}
		if seen[row.name] {
			t.Errorf("row %q appears twice", row.name)
		}
		seen[row.name] = true
		// Every button with an icon shows it here, so the page and the bar match.
		// Ctrl-C/Ctrl-D are text buttons and have no icon to sample.
		if row.icon == nil && row.name != "Ctrl-C / Ctrl-D" {
			t.Errorf("row %q has no icon sample", row.name)
		}
	}

	joined := strings.ToLower(strings.Join(namesAndText(), " "))
	for _, want := range []string{"connection tree", "config panel", "activity log", "quick connect", "vault", "split", "clipboard", "delete"} {
		if !strings.Contains(joined, want) {
			t.Errorf("welcome help does not mention %q", want)
		}
	}
}

func namesAndText() []string {
	all := make([]string, 0, len(welcomeHelp)*2)
	for _, row := range welcomeHelp {
		all = append(all, row.name, row.what)
	}
	return all
}

// TestWelcomeSessionNotice_MatchesTheDeliveredState fails if the start page goes
// back to claiming sessions are unimplemented, which was wrong for many stages.
func TestWelcomeSessionNotice_MatchesTheDeliveredState(t *testing.T) {
	lower := strings.ToLower(welcomeSessionNotice)
	if strings.Contains(lower, "not implemented") {
		t.Errorf("notice claims sessions are unimplemented: %q", welcomeSessionNotice)
	}
	for _, protocol := range []string{"SSH", "RDP", "VNC", "Telnet"} {
		if !strings.Contains(welcomeSessionNotice, protocol) {
			t.Errorf("notice does not mention %s", protocol)
		}
	}
}

func TestWelcomeContent_BuildsWithTheHelpCard(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	if content := welcomeContent(); content == nil {
		t.Fatal("welcome content is nil")
	}
}
