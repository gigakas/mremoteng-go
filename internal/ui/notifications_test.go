package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestNotifications_Add_KeepsTheNewestWithinTheLimit(t *testing.T) {
	history := notifications{}
	for i := 0; i < notificationLimit+5; i++ {
		history.add(noticeError, "problem")
	}

	if got := history.count(); got != notificationLimit {
		t.Fatalf("count = %d, want the history bounded at %d", got, notificationLimit)
	}
	if history.entries[0].Level != noticeError {
		t.Errorf("level = %v, want the level kept", history.entries[0].Level)
	}
	if history.entries[0].At.IsZero() {
		t.Error("the entry has no timestamp")
	}

	history.add(noticeWarning, "   ")
	if got := history.count(); got != notificationLimit {
		t.Errorf("count = %d, want an empty message refused", got)
	}
}

func TestNotifications_Render_ListsOneLinePerMessage(t *testing.T) {
	history := notifications{}
	if got := history.render(); !strings.Contains(got, "No problems") {
		t.Errorf("empty render = %q, want a placeholder", got)
	}
	history.add(noticeError, "Could not list /home/tester:\npermission denied")
	history.add(noticeWarning, "RDP session closed: connection reset")

	rendered := history.render()
	lines := strings.Split(rendered, "\n")
	if len(lines) != 2 {
		t.Fatalf("render = %q, want one line per entry", rendered)
	}
	if !strings.Contains(lines[0], "error") || !strings.Contains(lines[0], "permission denied") {
		t.Errorf("first line = %q, want the error and its single-line text", lines[0])
	}
	if !strings.Contains(lines[1], "warning") || !strings.Contains(lines[1], "connection reset") {
		t.Errorf("second line = %q, want the warning", lines[1])
	}
	if strings.Contains(rendered, "\n\n") {
		t.Errorf("render = %q, want no blank line from the multi-line message", rendered)
	}
}

// TestSetProblem_RecordsANotification is the contract that makes the panel worth
// having: whatever the status bar says and then replaces is kept.
func TestSetProblem_RecordsANotification(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})

	s.setProblem("Could not read saved connections")

	if got := s.notifications.count(); got != 1 {
		t.Fatalf("notifications = %d, want the problem recorded", got)
	}
	if !strings.Contains(s.notificationPanel.text.Text, "Could not read saved connections") {
		t.Errorf("panel text = %q, want the problem", s.notificationPanel.text.Text)
	}
	if s.notificationsButton.Text != "1" {
		t.Errorf("badge = %q, want the waiting count", s.notificationsButton.Text)
	}
	if !strings.Contains(s.notificationPanel.title.Text, "(1)") {
		t.Errorf("title = %q, want the count", s.notificationPanel.title.Text)
	}
}

func TestSetWarning_RecordsAWarningWithoutDangerStyling(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})

	s.setWarning("RDP session closed: connection reset")

	if s.notifications.count() != 1 || s.notifications.entries[0].Level != noticeWarning {
		t.Errorf("entries = %+v, want one warning", s.notifications.entries)
	}
	if !strings.Contains(s.status.Text, "connection reset") {
		t.Errorf("status = %q, want the warning shown", s.status.Text)
	}
}

func TestNotificationPanel_Clear_EmptiesTheHistoryAndTheBadge(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	s.setProblem("first")
	s.setProblem("second")

	test.Tap(s.notificationPanel.clear)

	if s.notifications.count() != 0 {
		t.Errorf("notifications = %d, want none", s.notifications.count())
	}
	if s.notificationsButton.Text != "" {
		t.Errorf("badge = %q, want it cleared", s.notificationsButton.Text)
	}
	if !strings.Contains(s.notificationPanel.text.Text, "No problems") {
		t.Errorf("panel text = %q, want the empty placeholder", s.notificationPanel.text.Text)
	}
}

// TestToggleNotifications_ReplacesTheLog keeps the bottom slot single: the two panels
// never eat the terminal's height together.
func TestToggleNotifications_ReplacesTheLog(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})

	s.toggleLog()
	if !s.logVisible || s.notificationPanel.box.Visible() {
		t.Fatal("the log did not open on its own")
	}
	s.toggleNotifications()
	if s.logVisible || !s.notificationsVisible {
		t.Error("the notifications panel did not replace the log")
	}
	if s.logConsole.box.Visible() {
		t.Error("the log is still visible behind the notifications panel")
	}
	s.toggleLog()
	if !s.logVisible || s.notificationsVisible {
		t.Error("the log did not replace the notifications panel")
	}
}

func TestNotificationsHint_CountsTheWaitingProblems(t *testing.T) {
	if got := notificationsHint(0); strings.Contains(got, "1 ") {
		t.Errorf("hint = %q, want no count phrase when empty", got)
	}
	if got := notificationsHint(1); !strings.Contains(got, "1 problem") {
		t.Errorf("hint(1) = %q, want the singular", got)
	}
	if got := notificationsHint(4); !strings.Contains(got, "4 problems") {
		t.Errorf("hint(4) = %q, want the plural", got)
	}
}
