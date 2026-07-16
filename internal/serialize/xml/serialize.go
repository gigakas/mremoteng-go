package xml

import (
	"bytes"
	stdxml "encoding/xml"
	"errors"
	"fmt"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

var ErrInvalidDocument = errors.New("xml: invalid connection document")

// SerializeOptions controls v2.8 output. Zero KDFIterations selects the
// mRemoteNG-compatible default of 1000.
type SerializeOptions struct {
	Password           []byte
	KDFIterations      int
	FullFileEncryption bool
	Export             bool
	SaveFilter         *SaveFilter
}

// SaveFilter controls credential and inheritance disclosure. A nil filter
// saves all fields, matching normal mRemoteNG saves.
type SaveFilter struct {
	Username    bool
	Domain      bool
	Password    bool
	Inheritance bool
}

// Serialize writes the latest supported format (ConfVersion 2.8).
func Serialize(document *Document, options SerializeOptions) ([]byte, error) {
	if document == nil || document.Root == nil {
		return nil, ErrInvalidDocument
	}
	iterations := options.KDFIterations
	if iterations == 0 {
		iterations = 1000
	}
	provider, err := security.NewAEADWithIterations(iterations)
	if err != nil {
		return nil, fmt.Errorf("xml: KdfIterations: %w", err)
	}
	password := options.Password
	if len(password) == 0 {
		password = []byte(defaultPassword)
	}
	marker := "ThisIsProtected"
	if bytes.Equal(password, []byte(defaultPassword)) {
		marker = "ThisIsNotProtected"
	}
	protected, err := provider.Encrypt(marker, password)
	if err != nil {
		return nil, fmt.Errorf("xml: encrypt protection marker: %w", err)
	}

	serializer := nodeSerializer{provider: provider, password: password, filter: normalizedSaveFilter(options.SaveFilter)}
	inner, err := serializer.encodeChildren(document.Root)
	if err != nil {
		return nil, err
	}
	if options.FullFileEncryption {
		ciphertext, err := provider.Encrypt(string(inner), password)
		if err != nil {
			return nil, fmt.Errorf("xml: encrypt full-file payload: %w", err)
		}
		inner = []byte(ciphertext)
	}

	var output bytes.Buffer
	output.WriteString(stdxml.Header)
	encoder := stdxml.NewEncoder(&output)
	encoder.Indent("", "  ")
	root := stdxml.StartElement{
		Name: stdxml.Name{Space: "http://mremoteng.org", Local: "Connections"},
		Attr: []stdxml.Attr{
			stringXMLAttr("Name", document.Root.Base().Raw.Name),
			boolXMLAttr("Export", options.Export),
			stringXMLAttr("EncryptionEngine", "AES"),
			stringXMLAttr("BlockCipherMode", "GCM"),
			intXMLAttr("KdfIterations", iterations),
			boolXMLAttr("FullFileEncryption", options.FullFileEncryption),
			stringXMLAttr("Protected", protected),
			stringXMLAttr("ConfVersion", "2.8"),
		},
	}
	if err := encoder.EncodeToken(root); err != nil {
		return nil, fmt.Errorf("xml: encode root: %w", err)
	}
	if options.FullFileEncryption {
		if err := encoder.EncodeToken(stdxml.CharData(inner)); err != nil {
			return nil, fmt.Errorf("xml: encode encrypted payload: %w", err)
		}
	} else {
		if err := encoder.Flush(); err != nil {
			return nil, fmt.Errorf("xml: flush root: %w", err)
		}
		output.Write(inner)
	}
	if err := encoder.EncodeToken(root.End()); err != nil {
		return nil, fmt.Errorf("xml: close root: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("xml: flush document: %w", err)
	}
	return output.Bytes(), nil
}

func normalizedSaveFilter(filter *SaveFilter) SaveFilter {
	if filter == nil {
		return SaveFilter{Username: true, Domain: true, Password: true, Inheritance: true}
	}
	return *filter
}

func stringXMLAttr(name, value string) stdxml.Attr {
	return stdxml.Attr{Name: stdxml.Name{Local: name}, Value: value}
}

func boolXMLAttr(name string, value bool) stdxml.Attr {
	if value {
		return stringXMLAttr(name, "true")
	}
	return stringXMLAttr(name, "false")
}

func intXMLAttr(name string, value int) stdxml.Attr {
	return stringXMLAttr(name, fmt.Sprintf("%d", value))
}

func enumXMLAttr[T ~string](name string, value T) stdxml.Attr {
	return stringXMLAttr(name, string(value))
}

func nodeTypeAttr(node connection.Node) stdxml.Attr {
	return enumXMLAttr("Type", node.Kind())
}
