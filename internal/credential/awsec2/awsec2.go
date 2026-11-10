// Package awsec2 implements connection.AddressProviderAmazonWebServices:
// resolving a connection's Hostname from its EC2InstanceID/EC2Region by
// calling EC2's DescribeInstances API directly. It signs requests with
// AWS Signature Version 4 by hand (crypto/hmac + crypto/sha256) rather
// than taking a dependency on the AWS SDK, per the project's "no new
// external dependencies without justification, prefer the standard
// library" rule -- SigV4 for a single, fixed query-API call is a
// well-specified, self-contained algorithm, not a case that needs the
// SDK's broader service coverage.
package awsec2

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const service = "ec2"
const apiVersion = "2016-11-15"

// Client signs and sends EC2 Query API requests with a fixed set of
// long-term or temporary credentials.
type Client struct {
	// HTTPClient defaults to http.DefaultClient if nil.
	HTTPClient *http.Client

	AccessKeyID     string
	SecretAccessKey string
	// SessionToken is optional, needed only for temporary (STS-issued)
	// credentials.
	SessionToken string

	// Endpoint overrides the request target for testing; production
	// callers leave it empty and InstanceAddress builds the real
	// "https://ec2.{region}.amazonaws.com/" endpoint for whatever region
	// is passed in.
	Endpoint string

	// now defaults to time.Now; overridable so signature tests can use a
	// fixed clock.
	now func() time.Time
}

func New(accessKeyID, secretAccessKey string) *Client {
	return &Client{AccessKeyID: accessKeyID, SecretAccessKey: secretAccessKey}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *Client) endpoint(region string) string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	return fmt.Sprintf("https://ec2.%s.amazonaws.com/", region)
}

// InstanceAddress resolves instanceID's address in region: the public IP
// if EC2 assigned one, otherwise the private IP -- matching mRemoteNG's
// own preference of "connect via whatever address is actually reachable,
// preferring public" for an instance that may or may not have a public
// IP depending on its VPC/subnet configuration.
func (c *Client) InstanceAddress(ctx context.Context, region, instanceID string) (string, error) {
	params := url.Values{
		"Action":       {"DescribeInstances"},
		"Version":      {apiVersion},
		"InstanceId.1": {instanceID},
	}

	body, err := c.doSigned(ctx, region, params)
	if err != nil {
		return "", err
	}

	var parsed describeInstancesResponse
	if err := xml.Unmarshal(body, &parsed); err != nil {
		if errResp, ok := parseErrorResponse(body); ok {
			return "", fmt.Errorf("awsec2: DescribeInstances %s: %s: %s", instanceID, errResp.Code, errResp.Message)
		}
		return "", fmt.Errorf("awsec2: parse DescribeInstances response for %s: %w", instanceID, err)
	}

	for _, reservation := range parsed.ReservationSet.Items {
		for _, instance := range reservation.InstancesSet.Items {
			if instance.PublicIP != "" {
				return instance.PublicIP, nil
			}
			if instance.PrivateIP != "" {
				return instance.PrivateIP, nil
			}
		}
	}
	return "", fmt.Errorf("awsec2: instance %s not found in region %s (or has no address)", instanceID, region)
}

type describeInstancesResponse struct {
	XMLName        xml.Name `xml:"DescribeInstancesResponse"`
	ReservationSet struct {
		Items []struct {
			InstancesSet struct {
				Items []struct {
					InstanceID string `xml:"instanceId"`
					PrivateIP  string `xml:"privateIpAddress"`
					PublicIP   string `xml:"ipAddress"`
				} `xml:"item"`
			} `xml:"instancesSet"`
		} `xml:"item"`
	} `xml:"reservationSet"`
}

type awsErrorResponse struct {
	XMLName xml.Name `xml:"Response"`
	Errors  struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	} `xml:"Errors"`
}

func parseErrorResponse(body []byte) (struct{ Code, Message string }, bool) {
	var parsed awsErrorResponse
	if err := xml.Unmarshal(body, &parsed); err != nil || parsed.Errors.Error.Code == "" {
		return struct{ Code, Message string }{}, false
	}
	return struct{ Code, Message string }{parsed.Errors.Error.Code, parsed.Errors.Error.Message}, true
}

// doSigned sends a SigV4-signed GET request to region's EC2 endpoint
// with the given query parameters and returns the raw response body.
func (c *Client) doSigned(ctx context.Context, region string, params url.Values) ([]byte, error) {
	endpoint := c.endpoint(region)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("awsec2: parse endpoint %q: %w", endpoint, err)
	}
	if u.Path == "" {
		u.Path = "/"
	}

	now := c.clock().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	canonicalQuery := canonicalQueryString(params)
	u.RawQuery = canonicalQuery

	headers := map[string]string{
		"host":       u.Host,
		"x-amz-date": amzDate,
	}
	if c.SessionToken != "" {
		headers["x-amz-security-token"] = c.SessionToken
	}
	signedHeaderNames := sortedKeys(headers)
	canonicalHeaders := ""
	for _, name := range signedHeaderNames {
		canonicalHeaders += name + ":" + headers[name] + "\n"
	}
	signedHeaders := strings.Join(signedHeaderNames, ";")

	payloadHash := sha256Hex(nil)
	canonicalRequest := strings.Join([]string{
		http.MethodGet,
		u.Path,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := signingKey(c.SecretAccessKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authorization := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.AccessKeyID, credentialScope, signedHeaders, signature)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("awsec2: build request: %w", err)
	}
	req.Header.Set("Host", u.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	if c.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", c.SessionToken)
	}
	req.Header.Set("Authorization", authorization)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("awsec2: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("awsec2: read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if errResp, ok := parseErrorResponse(body); ok {
			return nil, fmt.Errorf("awsec2: HTTP %d: %s: %s", resp.StatusCode, errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("awsec2: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// canonicalQueryString builds a SigV4 canonical query string: params
// sorted by key, each key and value percent-encoded per RFC 3986 (which
// url.Values.Encode already does, and also already sorts by key).
func canonicalQueryString(params url.Values) string {
	return params.Encode()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func signingKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}
