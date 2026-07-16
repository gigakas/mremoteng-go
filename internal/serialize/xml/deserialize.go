// Package xml reads and writes mRemoteNG connection XML files.
package xml

import (
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

const defaultPassword = "mR3m"

var (
	ErrEmptyDocument      = errors.New("xml: empty connection document")
	ErrMalformedXML       = errors.New("xml: malformed connection document")
	ErrUnsupportedVersion = errors.New("xml: unsupported ConfVersion")
	ErrUnsupportedCipher  = errors.New("xml: unsupported cipher")
	ErrAuthentication     = errors.New("xml: authentication failed")
)

// Metadata contains root attributes that are not connection-tree values.
type Metadata struct {
	Name               string
	Export             bool
	EncryptionEngine   string
	BlockCipherMode    string
	KDFIterations      int
	FullFileEncryption bool
	Protected          string
	ConfVersion        string
}

// Document is a decoded mRemoteNG connection file.
type Document struct {
	Root     *connection.ContainerInfo
	Metadata Metadata
}

// Options controls decryption. An empty password uses mRemoteNG's default
// connection-file password.
type Options struct {
	Password []byte
}

type rootEnvelope struct {
	XMLName stdxml.Name
	Attrs   []stdxml.Attr `xml:",any,attr"`
	Nodes   []nodeElement `xml:"Node"`
	Text    string        `xml:",chardata"`
}

type nodeElement struct {
	Attrs    []stdxml.Attr `xml:",any,attr"`
	Children []nodeElement `xml:"Node"`
}

type nodeWrapper struct {
	Nodes []nodeElement `xml:"Node"`
}

// Deserialize parses versions 2.6, 2.7 and 2.8 and reconstructs the ordered
// connection/container tree.
func Deserialize(data []byte, options Options) (*Document, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, ErrEmptyDocument
	}
	var envelope rootEnvelope
	if err := stdxml.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedXML, err)
	}
	if envelope.XMLName.Local != "Connections" {
		return nil, fmt.Errorf("%w: root element %q", ErrMalformedXML, envelope.XMLName.Local)
	}

	attrs := makeAttributes(envelope.Attrs)
	version, normalizedVersion, err := parseVersion(attrs.string("ConfVersion"))
	if err != nil {
		return nil, err
	}
	metadata := Metadata{
		Name:               strings.TrimSpace(attrs.string("Name")),
		Export:             attrs.boolean("Export"),
		EncryptionEngine:   defaultString(attrs.string("EncryptionEngine"), "AES"),
		BlockCipherMode:    defaultString(attrs.string("BlockCipherMode"), "GCM"),
		KDFIterations:      attrs.integer("KdfIterations"),
		FullFileEncryption: attrs.boolean("FullFileEncryption"),
		Protected:          attrs.string("Protected"),
		ConfVersion:        normalizedVersion,
	}
	if !strings.EqualFold(metadata.EncryptionEngine, "AES") || !strings.EqualFold(metadata.BlockCipherMode, "GCM") {
		return nil, fmt.Errorf("%w: %s/%s", ErrUnsupportedCipher, metadata.EncryptionEngine, metadata.BlockCipherMode)
	}
	provider, err := security.NewAEADWithIterations(metadata.KDFIterations)
	if err != nil {
		return nil, fmt.Errorf("xml: KdfIterations: %w", err)
	}
	password := options.Password
	if len(password) == 0 {
		password = []byte(defaultPassword)
	}
	if metadata.Protected == "" {
		return nil, fmt.Errorf("%w: missing Protected attribute", ErrAuthentication)
	}
	if _, err := provider.Decrypt(compactCiphertext(metadata.Protected), password); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthentication, err)
	}

	nodes := envelope.Nodes
	if metadata.FullFileEncryption {
		plaintext, err := provider.Decrypt(compactCiphertext(envelope.Text), password)
		if err != nil {
			return nil, fmt.Errorf("%w: full-file payload: %v", ErrAuthentication, err)
		}
		var wrapper nodeWrapper
		wrapped := []byte("<Wrapper>" + plaintext + "</Wrapper>")
		if err := stdxml.Unmarshal(wrapped, &wrapper); err != nil {
			return nil, fmt.Errorf("%w: decrypted payload: %v", ErrMalformedXML, err)
		}
		nodes = wrapper.Nodes
	}

	root, err := connection.NewRootInfo()
	if err != nil {
		return nil, fmt.Errorf("xml: create root: %w", err)
	}
	root.Base().Raw.Name = metadata.Name
	decoder := nodeDecoder{version: version, provider: provider, password: password}
	for i := range nodes {
		if err := decoder.addNode(root, nodes[i]); err != nil {
			return nil, err
		}
	}
	return &Document{Root: root, Metadata: metadata}, nil
}

func parseVersion(raw string) (int, string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), ",", ".")
	parsed, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, normalized, fmt.Errorf("%w: %q", ErrUnsupportedVersion, raw)
	}
	switch parsed {
	case 2.6:
		return 26, "2.6", nil
	case 2.7:
		return 27, "2.7", nil
	case 2.8:
		return 28, "2.8", nil
	default:
		return 0, normalized, fmt.Errorf("%w: %q", ErrUnsupportedVersion, raw)
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func compactCiphertext(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}
