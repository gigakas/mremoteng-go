package connection

import (
	"errors"
	"strings"
	"testing"
)

func TestNewConnectionInfo_Defaults_MatchMRemoteNG(t *testing.T) {
	connection, err := NewConnectionInfo()
	if err != nil {
		t.Fatal(err)
	}
	if connection.ID() == "" {
		t.Error("generated ID is empty")
	}
	if connection.Raw.Name != "New Connection" || connection.Raw.Protocol != ProtocolRDP {
		t.Errorf("identity defaults = (%q,%q), want New Connection/RDP", connection.Raw.Name, connection.Raw.Protocol)
	}
	if connection.Raw.Port != 3389 || connection.Raw.RDPVersion != RDPVersion10 {
		t.Errorf("RDP defaults = port %d/version %q, want 3389/Rdc10", connection.Raw.Port, connection.Raw.RDPVersion)
	}
	if !connection.Raw.UseCredSsp || !connection.Raw.AutomaticResize {
		t.Error("UseCredSsp and AutomaticResize must default true")
	}
	if connection.Raw.Resolution != RDPResolutionSmartSize || connection.Raw.Colors != RDPColors16Bit {
		t.Errorf("appearance defaults = (%q,%q)", connection.Raw.Resolution, connection.Raw.Colors)
	}
	if connection.Raw.VNCCompression != VNCCompressionNone || connection.Raw.VNCEncoding != VNCEncodingHextile {
		t.Errorf("VNC defaults = (%q,%q)", connection.Raw.VNCCompression, connection.Raw.VNCEncoding)
	}
}

func TestNewConnectionInfoWithID_ExistingID_PreservesValue(t *testing.T) {
	const id = "existing-id"
	connection, err := NewConnectionInfoWithID(id)
	if err != nil {
		t.Fatal(err)
	}
	if connection.ID() != id {
		t.Errorf("ID = %q, want %q", connection.ID(), id)
	}
}

func TestNewConnectionInfoWithID_EmptyID_ReturnsError(t *testing.T) {
	if _, err := NewConnectionInfoWithID("  "); !errors.Is(err, ErrInvalidID) {
		t.Errorf("error = %v, want ErrInvalidID", err)
	}
}

func TestNewConnectionInfo_GeneratedID_IsRFC4122Version4(t *testing.T) {
	connection, err := NewConnectionInfo()
	if err != nil {
		t.Fatal(err)
	}
	id := connection.ID()
	if len(id) != 36 || strings.Count(id, "-") != 4 || id[14] != '4' {
		t.Errorf("ID = %q, want RFC 4122 version 4 shape", id)
	}
}

func TestDefaultPort_Protocols_ReturnExpectedValues(t *testing.T) {
	cases := map[ProtocolType]int{
		ProtocolRDP:        3389,
		ProtocolVNC:        5900,
		ProtocolSSH2:       22,
		ProtocolTelnet:     23,
		ProtocolRlogin:     513,
		ProtocolHTTP:       80,
		ProtocolHTTPS:      443,
		ProtocolPowerShell: 5985,
		ProtocolSerial:     9600,
		ProtocolAnyDesk:    0,
	}
	for protocol, want := range cases {
		if got := DefaultPort(protocol); got != want {
			t.Errorf("DefaultPort(%q) = %d, want %d", protocol, got, want)
		}
	}
}

func TestNewContainerInfo_Defaults_AreFolderDefaults(t *testing.T) {
	container, err := NewContainerInfo()
	if err != nil {
		t.Fatal(err)
	}
	if container.Kind() != NodeKindContainer || container.Base().Raw.Name != "New Folder" || !container.Expanded() {
		t.Errorf("container defaults = kind %q, name %q, expanded %v", container.Kind(), container.Base().Raw.Name, container.Expanded())
	}
	if container.Base().Raw.Protocol != ProtocolRDP || container.Base().Raw.Port != 3389 {
		t.Error("container must retain the complete connection-property defaults")
	}
}
