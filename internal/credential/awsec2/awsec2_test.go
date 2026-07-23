package awsec2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const describeInstancesXML = `<?xml version="1.0" encoding="UTF-8"?>
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-0123456789abcdef0</instanceId>
          <privateIpAddress>10.0.1.23</privateIpAddress>
          <ipAddress>203.0.113.10</ipAddress>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`

const describeInstancesXMLPrivateOnly = `<?xml version="1.0" encoding="UTF-8"?>
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-privateonly</instanceId>
          <privateIpAddress>10.0.1.99</privateIpAddress>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`

const errorXML = `<?xml version="1.0" encoding="UTF-8"?>
<Response>
  <Errors>
    <Error>
      <Code>InvalidInstanceID.NotFound</Code>
      <Message>The instance ID 'i-doesnotexist' does not exist</Message>
    </Error>
  </Errors>
  <RequestID>abc-123</RequestID>
</Response>`

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &Client{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secret",
		Endpoint:        srv.URL,
		now:             func() time.Time { return time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC) },
	}
}

func TestInstanceAddress_PrefersPublicIPWhenPresent(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(describeInstancesXML))
	})

	got, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-0123456789abcdef0")
	if err != nil {
		t.Fatalf("InstanceAddress: %v", err)
	}
	if got != "203.0.113.10" {
		t.Errorf("InstanceAddress = %q, want the public IP %q", got, "203.0.113.10")
	}
}

func TestInstanceAddress_FallsBackToPrivateIPWhenNoPublicIP(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(describeInstancesXMLPrivateOnly))
	})

	got, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-privateonly")
	if err != nil {
		t.Fatalf("InstanceAddress: %v", err)
	}
	if got != "10.0.1.99" {
		t.Errorf("InstanceAddress = %q, want the private IP %q", got, "10.0.1.99")
	}
}

func TestInstanceAddress_AWSErrorResponse_ReturnsErrorWithCodeAndMessage(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errorXML))
	})

	_, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-doesnotexist")
	if err == nil {
		t.Fatal("expected an error for an AWS error response")
	}
	if !strings.Contains(err.Error(), "InvalidInstanceID.NotFound") {
		t.Errorf("error = %v, want it to mention the AWS error code", err)
	}
}

func TestInstanceAddress_RequestIsSigV4Signed(t *testing.T) {
	var gotAuth, gotAmzDate, gotQuery string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAmzDate = r.Header.Get("X-Amz-Date")
		gotQuery = r.URL.RawQuery
		w.Write([]byte(describeInstancesXML))
	})

	if _, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-0123456789abcdef0"); err != nil {
		t.Fatalf("InstanceAddress: %v", err)
	}

	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256 Credential=AKIAEXAMPLE/20260724/eu-central-1/ec2/aws4_request, SignedHeaders=host;x-amz-date, Signature=") {
		t.Errorf("Authorization header = %q, want an AWS4-HMAC-SHA256 credential scope for eu-central-1/ec2/aws4_request", gotAuth)
	}
	if gotAmzDate != "20260724T120000Z" {
		t.Errorf("X-Amz-Date = %q, want %q", gotAmzDate, "20260724T120000Z")
	}
	if !strings.Contains(gotQuery, "Action=DescribeInstances") || !strings.Contains(gotQuery, "InstanceId.1=i-0123456789abcdef0") {
		t.Errorf("query string = %q, missing expected DescribeInstances params", gotQuery)
	}
}

func TestInstanceAddress_SessionToken_IsSignedAndSent(t *testing.T) {
	var gotToken, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Amz-Security-Token")
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(describeInstancesXML))
	}))
	defer srv.Close()

	c := &Client{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secret",
		SessionToken:    "session-token-value",
		Endpoint:        srv.URL,
		now:             func() time.Time { return time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC) },
	}

	if _, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-0123456789abcdef0"); err != nil {
		t.Fatalf("InstanceAddress: %v", err)
	}
	if gotToken != "session-token-value" {
		t.Errorf("X-Amz-Security-Token = %q, want %q", gotToken, "session-token-value")
	}
	if !strings.Contains(gotAuth, "SignedHeaders=host;x-amz-date;x-amz-security-token") {
		t.Errorf("Authorization = %q, want x-amz-security-token included in SignedHeaders", gotAuth)
	}
}

func TestInstanceAddress_InstanceNotInResponse_ReturnsError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><DescribeInstancesResponse><reservationSet></reservationSet></DescribeInstancesResponse>`))
	})

	if _, err := c.InstanceAddress(context.Background(), "eu-central-1", "i-missing"); err == nil {
		t.Fatal("expected an error when no reservations are returned")
	}
}
