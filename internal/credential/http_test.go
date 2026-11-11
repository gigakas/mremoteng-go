package credential_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

func TestHTTPClient_NilGiven_ReturnsDefaultClient(t *testing.T) {
	if got := credential.HTTPClient(nil); got != http.DefaultClient {
		t.Errorf("HTTPClient(nil) = %p, want http.DefaultClient %p", got, http.DefaultClient)
	}
}

func TestHTTPClient_NonNilGiven_ReturnsGivenClient(t *testing.T) {
	custom := &http.Client{}
	if got := credential.HTTPClient(custom); got != custom {
		t.Errorf("HTTPClient(custom) = %p, want %p", got, custom)
	}
}

func TestInsecureHTTPClient_Transport_SkipsCertificateVerification(t *testing.T) {
	c := credential.InsecureHTTPClient()
	if c == http.DefaultClient {
		t.Fatal("InsecureHTTPClient returned http.DefaultClient, which verifies certificates")
	}
	transport, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", c.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("TLSClientConfig.InsecureSkipVerify = false, want true")
	}
}

type failingTransport struct{ err error }

func (t failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

// erroringBody stands in for a connection that dies mid-body: Read always
// fails.
type erroringBody struct{}

func (erroringBody) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (erroringBody) Close() error             { return nil }

func TestDo_TransportFailure_WrapsTransportContext(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/secret", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	_, _, err = credential.Do(&http.Client{Transport: failingTransport{err: errors.New("dial failed")}}, req, "provider: request", "provider: read")
	if err == nil {
		t.Fatal("expected an error from a failing transport")
	}
	if want := "provider: request: "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want prefix %q", err, want)
	}
}

func TestDo_BodyReadFailure_WrapsReadContext(t *testing.T) {
	server := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: erroringBody{}}, nil
	})}
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid/secret", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	_, _, err = credential.Do(server, req, "provider: request", "provider: read")
	if err == nil {
		t.Fatal("expected an error when the response body cannot be read")
	}
	if want := "provider: read: "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want prefix %q", err, want)
	}
}

func TestDo_OKResponse_ReturnsBodyAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	body, status, err := credential.Do(http.DefaultClient, req, "provider: request", "provider: read")
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", body, `{"ok":true}`)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestCheckStatus_WantStatus_ReturnsNil(t *testing.T) {
	if err := credential.CheckStatus(http.StatusOK, http.StatusOK, []byte(`{}`), "provider: get"); err != nil {
		t.Errorf("CheckStatus(200, want 200) = %v, want nil", err)
	}
}

func TestCheckStatus_OtherStatus_ReportsContextStatusAndBody(t *testing.T) {
	err := credential.CheckStatus(http.StatusUnauthorized, http.StatusOK, []byte("Invalid API Key"), "provider: get password 7")
	if err == nil {
		t.Fatal("expected an error for a non-matching status")
	}
	want := "provider: get password 7: HTTP 401: Invalid API Key"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestDecodeJSON_InvalidBody_WrapsContext(t *testing.T) {
	var out struct{}
	err := credential.DecodeJSON([]byte("not json"), &out, "provider: parse response")
	if err == nil {
		t.Fatal("expected an error for an invalid JSON body")
	}
	if want := "provider: parse response: "; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want prefix %q", err, want)
	}
}
