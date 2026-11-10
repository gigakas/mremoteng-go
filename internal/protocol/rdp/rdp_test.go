<<<<<<< HEAD
package rdp

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArguments_DefaultTarget_UsesPortAndStdinCredentials(t *testing.T) {
	args := Arguments(Target{Host: "desk.example.com", Username: "kas", Domain: "CORP"}, Options{Password: "secret"})
	line := CommandLine(args)
	for _, want := range []string{"/v:desk.example.com:3389", "/u:kas", "/d:CORP", "/from-stdin:force", "/cert:tofu"} {
		if !strings.Contains(line, want) {
			t.Errorf("arguments = %q, want %q", line, want)
		}
	}
	if strings.Contains(line, "secret") {
		t.Errorf("arguments leak the password: %q", line)
	}
}

func TestArguments_EmbeddedAndAuthOnly_AddExpectedFlags(t *testing.T) {
	embedded := CommandLine(Arguments(Target{Host: "h", Port: 3390}, Options{ParentWindowID: 42}))
	if !strings.Contains(embedded, "/parent-window:42") || !strings.Contains(embedded, "+smart-sizing") {
		t.Errorf("embedded arguments = %q", embedded)
	}
	auth := CommandLine(Arguments(Target{Host: "h"}, Options{AuthOnly: true}))
	if !strings.Contains(auth, "+auth-only") {
		t.Errorf("auth-only arguments = %q", auth)
	}
	detached := CommandLine(Arguments(Target{Host: "h"}, Options{Width: 1280, Height: 800}))
	if !strings.Contains(detached, "/size:1280x800") {
		t.Errorf("detached arguments = %q", detached)
	}
}

func TestArguments_DeviceRedirectionAndDisplayFlags(t *testing.T) {
	args := CommandLine(Arguments(Target{Host: "h"}, Options{
		Clipboard:         true,
		Printers:          true,
		Audio:             true,
		Drives:            []string{"shared:/home/user/docs", " "},
		DynamicResolution: true,
		MultiMonitor:      true,
	}))
	for _, want := range []string{"+clipboard", "/printer", "/audio-mode:0", "/drive:shared:/home/user/docs", "/dynamic-resolution", "/multimon"} {
		if !strings.Contains(args, want) {
			t.Errorf("arguments = %q, missing %q", args, want)
		}
	}
	plain := CommandLine(Arguments(Target{Host: "h"}, Options{}))
	for _, unwanted := range []string{"+clipboard", "/printer", "/dynamic-resolution", "/multimon", "/drive:"} {
		if strings.Contains(plain, unwanted) {
			t.Errorf("default arguments = %q, want no %q", plain, unwanted)
		}
	}
}

func TestArguments_Gateway_AddsHostAndCredentials(t *testing.T) {
	args := CommandLine(Arguments(Target{Host: "desk"}, Options{Gateway: Gateway{Host: "gw.example.com", Username: "gwuser", Domain: "CORP"}}))
	for _, want := range []string{"/g:gw.example.com", "/gu:gwuser", "/gd:CORP"} {
		if !strings.Contains(args, want) {
			t.Errorf("arguments = %q, missing %q", args, want)
		}
	}
	if strings.Contains(CommandLine(Arguments(Target{Host: "desk"}, Options{})), "/g:") {
		t.Error("default arguments contain a gateway")
	}
}

func TestArguments_CertificatePolicy_DefaultsToToFU(t *testing.T) {
	if args := CommandLine(Arguments(Target{Host: "h"}, Options{})); !strings.Contains(args, "/cert:tofu") {
		t.Errorf("arguments = %q, want the tofu certificate policy", args)
	}
	ignored := CommandLine(Arguments(Target{Host: "h"}, Options{IgnoreCertificate: true}))
	if !strings.Contains(ignored, "/cert:ignore") || strings.Contains(ignored, "/cert:tofu") {
		t.Errorf("arguments = %q, want only /cert:ignore", ignored)
	}
}

func TestArguments_SecurityOverride_IsExplicit(t *testing.T) {
	args := CommandLine(Arguments(Target{Host: "h"}, Options{Security: "rdp"}))
	if !strings.Contains(args, "/sec:rdp") {
		t.Errorf("arguments = %q, want /sec:rdp", args)
	}
	defaultArgs := CommandLine(Arguments(Target{Host: "h"}, Options{}))
	if strings.Contains(defaultArgs, "/sec:") {
		t.Errorf("default arguments force a security mode: %q", defaultArgs)
	}
}

// TestProbe_LocalXRDP exercises a real xrdp server. Set RDP_LOCAL to the
// endpoint and credentials, for example:
//
//	RDP_LOCAL=127.0.0.1:13389 RDP_LOCAL_USER=rdptest \
//	RDP_LOCAL_PASSWORD=rdptest go test ./internal/protocol/rdp/ -run LocalXRDP
func TestProbe_LocalXRDP(t *testing.T) {
	endpoint := os.Getenv("RDP_LOCAL")
	if endpoint == "" {
		t.Skip("RDP_LOCAL not set; no local xrdp server to probe")
	}
	host, portText, err := net.SplitHostPort(endpoint)
	if err != nil {
		t.Fatalf("RDP_LOCAL: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("RDP_LOCAL port: %v", err)
	}
	output, err := Probe(Target{
		Host:     host,
		Port:     port,
		Username: os.Getenv("RDP_LOCAL_USER"),
	}, Options{
		Password: os.Getenv("RDP_LOCAL_PASSWORD"),
		Security: os.Getenv("RDP_LOCAL_SECURITY"),
	})
	if err != nil {
		t.Fatalf("Probe: %v\noutput: %s", err, output)
	}
	t.Logf("authenticated against %s", endpoint)
}

func TestSummarize_PicksTheActionableLine(t *testing.T) {
	output := `[21:26:49:481] [1] [WARN][com.freerdp.crypto] - [verify_cb]: Certificate verification failure 'self-signed certificate (18)'
[21:26:49:482] [1] [ERROR][com.freerdp.crypto] - [tls_print_certificate_error]: @ WARNING: NEW HOST IDENTIFICATION! @
[21:26:49:482] [1] [ERROR][com.freerdp.crypto] - [tls_print_certificate_error]: Please contact your system administrator.
[21:26:49:482] [1] [ERROR][com.freerdp.core] - [freerdp_connect_begin]: Authentication only, exit status 0
`
	summary := Summarize(output)
	if !strings.Contains(summary, "certificate") {
		t.Errorf("summary = %q, want the certificate cause", summary)
	}
	if strings.Contains(summary, "\n") || len([]rune(summary)) > 260 {
		t.Errorf("summary is not a bounded line: %q", summary)
	}
	if got := Summarize(""); got != "no diagnostic output" {
		t.Errorf("Summarize(empty) = %q", got)
	}
}

func TestFindExecutable_ReturnsAvailableClient(t *testing.T) {
	path, err := FindExecutable()
	if err != nil {
		t.Skipf("no FreeRDP client on this machine: %v", err)
	}
	if !strings.Contains(filepath.Base(path), "freerdp") {
		t.Errorf("FindExecutable = %q, want a FreeRDP binary", path)
	}
}

// fakeClient writes the received argv and stdin to files so the launcher can be
// verified without a real RDP server.
func fakeClient(t *testing.T, exitCode int) (path, argsFile, stdinFile string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "fake-freerdp")
	argsFile = filepath.Join(dir, "args")
	stdinFile = filepath.Join(dir, "stdin")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat > " + stdinFile + "\nexit " + string(rune('0'+exitCode)) + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake client: %v", err)
	}
	return path, argsFile, stdinFile
}

func TestLaunch_PasswordGoesToStdinNotArgv(t *testing.T) {
	path, argsFile, stdinFile := fakeClient(t, 0)
	session, err := Launch(Target{Host: "127.0.0.1", Username: "demo"}, Options{Password: "s3cr3t", Executable: path})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	if strings.Contains(string(args), "s3cr3t") {
		t.Errorf("password leaked into argv: %q", args)
	}
	stdin, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("read stdin: %v", err)
	}
	if strings.TrimSpace(string(stdin)) != "s3cr3t" {
		t.Errorf("stdin = %q, want the password", stdin)
	}
}

func TestProbe_ClientFailure_ReturnsDiagnostics(t *testing.T) {
	path, _, _ := fakeClient(t, 1)
	output, err := Probe(Target{Host: "127.0.0.1"}, Options{Password: "x", Executable: path})
	if err == nil {
		t.Fatal("Probe succeeded with a failing client")
	}
	if output != "" {
		t.Logf("client output: %q", output)
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("error = %v, want an authentication failure", err)
	}
}

func TestLaunch_MissingExecutable_ReturnsError(t *testing.T) {
	_, err := Launch(Target{Host: "h"}, Options{Executable: "/nonexistent/xfreerdp"})
	if err == nil {
		t.Fatal("Launch succeeded with a missing client")
=======
package rdp_test

import (
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/protocol"
	_ "github.com/mRemoteNG/mremoteng-go/internal/protocol/rdp"
)

func newTestSession(t *testing.T, values func(*connection.ConnectionInfo)) (protocol.Protocol, error) {
	t.Helper()
	info, err := connection.NewConnectionInfo()
	if err != nil {
		t.Fatalf("NewConnectionInfo: %v", err)
	}
	info.Raw.Protocol = connection.ProtocolRDP
	values(info)
	return protocol.Create(info)
}

func TestNew_MissingHostname_ReturnsError(t *testing.T) {
	if _, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = ""
	}); err == nil {
		t.Fatal("expected an error for a missing hostname")
	}
}

func TestNew_ImplementsWindowProtocol(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "example.invalid"
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	win, ok := p.(protocol.WindowProtocol)
	if !ok {
		t.Fatalf("Create returned %T, want a protocol.WindowProtocol", p)
	}
	if h := win.NativeWindowHandle(); h != 0 {
		t.Errorf("NativeWindowHandle() = %x before Connect, want 0", h)
	}
}

// TestNew_DefaultPort exercises the DefaultPort(ProtocolRDP) fallback
// indirectly, through Connect's failure message, since buildArgs and the
// address it constructs are otherwise unexported. A missing FreeRDP
// binary is expected in a plain `go test` environment (no PATH
// modification here), so this only checks Connect fails for the right
// reason (client not found) rather than an address-construction panic —
// full argument-building coverage lives in the Windows-only
// TestEmbedChild_* / TestFindTopLevelForPID_* tests, which exercise a
// real launched process end-to-end.
func TestNew_DefaultPort_DoesNotPanicOnConnect(t *testing.T) {
	p, err := newTestSession(t, func(info *connection.ConnectionInfo) {
		info.Raw.Hostname = "example.invalid"
		info.Raw.Port = 0
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	err = p.Connect(t.Context())
	if err == nil {
		t.Skip("a FreeRDP client is actually installed and reachable in this environment; nothing to assert further here")
	}
	if !strings.Contains(err.Error(), "rdp:") {
		t.Errorf("Connect error = %q, want an rdp: prefixed error", err)
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
	}
}
