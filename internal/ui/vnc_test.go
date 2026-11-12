package ui

import (
	"fyne.io/fyne/v2"
	vncclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/vnc"
	"image"
	"image/color"
	"testing"
)

func TestSessionPanel_VNCConnect_ShowsFramebufferAndForwardsClicks(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	session := &fakeVNC{pixels: image.NewRGBA(image.Rect(0, 0, 100, 50))}
	session.pixels.SetRGBA(1, 1, color.RGBA{R: 0xff, A: 0xff})
	s.dialVNC = func(string, vncclient.Config) (vncSession, error) { return session, nil }
	profile := sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900)
	panel := s.newSessionPanel(profile)

	panel.startVNC(session)
	if panel.vncView == nil {
		t.Fatal("VNC view was not installed")
	}
	panel.vncView.Resize(fyne.NewSize(200, 100))
	panel.vncView.Tapped(&fyne.PointEvent{Position: fyne.NewPos(100, 50)})
	if len(session.pointer) < 2 {
		t.Fatalf("pointer events = %v, want press and release", session.pointer)
	}
	if session.pointer[0][0] < 45 || session.pointer[0][0] > 55 || session.pointer[0][1] < 20 || session.pointer[0][1] > 30 {
		t.Errorf("mapped click = %v, want around (50,25)", session.pointer[0])
	}
	panel.disconnectShell()
	if !session.closed {
		t.Error("disconnect did not close the VNC session")
	}
}

func TestProfile_Validate_AcceptsVNC(t *testing.T) {
	profile := sampleProfile(t, "Desktop", "", "VNC", "vnc.example.com", 5900)
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate(VNC) = %v", err)
	}
}
