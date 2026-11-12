package ui

import (
	"fyne.io/fyne/v2/test"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
	"image"
	"io"
	"testing"
)

type fakeStore struct {
	profiles  []profiles.Profile
	folders   []string
	defaults  []profiles.FolderDefault
	collapsed []string
	loadErr   error
	saveErr   error
	saves     int
}

type fakeConfirm struct {
	asked    []string
	accept   bool
	declines int
}

func newTestScreen(t *testing.T, store Store) *screen {
	t.Helper()
	app := test.NewApp()
	t.Cleanup(app.Quit)
	return newScreen(app, store)
}

func sampleProfile(t *testing.T, name, folder, protocol, host string, port int) profiles.Profile {
	t.Helper()
	id, err := profiles.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	return profiles.Profile{ID: id, Name: name, Folder: folder, Protocol: profiles.Protocol(protocol), Host: host, Port: port}
}

func fillConfig(s *screen, name, folder, host, port string) {
	s.props.name.SetText(name)
	s.props.folder.SetText(folder)
	s.props.host.SetText(host)
	s.props.port.SetText(port)
}

func sshPanelProfile(t *testing.T, username string) profiles.Profile {
	t.Helper()
	profile := sampleProfile(t, "Lab server", "", "SSH", "lab.example.com", 22)
	profile.Username = username
	return profile
}

type fakeShell struct {
	input  []byte
	output io.Reader
	closed bool
	sizes  []sshclient.TermSize
}

type fakeVNC struct {
	pixels  *image.RGBA
	pointer [][3]int
	closed  bool
	err     error
}

type fakeStream struct {
	written []byte
	closed  bool
}

type fakeClipboard struct{ content string }

type fakeRDPSession struct {
	closed bool
	done   chan struct{}
}

type fakeArea struct {
	id        uint64
	destroyed bool
	visible   bool
}
