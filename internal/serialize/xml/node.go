package xml

import (
	"fmt"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/security"
)

type nodeDecoder struct {
	version  int
	provider security.Provider
	password []byte
}

func (d nodeDecoder) addNode(parent *connection.ContainerInfo, element nodeElement) error {
	attrs := makeAttributes(element.Attrs)
	typeName := attrs.string("Type")
	if isIgnoredNodeType(typeName) {
		return nil
	}
	id := strings.TrimSpace(attrs.string("Id"))
	isContainer := strings.EqualFold(typeName, string(connection.NodeKindContainer))

	var node connection.Node
	var err error
	if isContainer {
		if id == "" {
			node, err = connection.NewContainerInfo()
		} else {
			node, err = connection.NewContainerInfoWithID(id)
		}
	} else {
		if id == "" {
			node, err = connection.NewConnectionInfo()
		} else {
			node, err = connection.NewConnectionInfoWithID(id)
		}
	}
	if err != nil {
		return fmt.Errorf("xml: create node %q: %w", attrs.string("Name"), err)
	}
	if err := d.decodeNode(node.Base(), attrs); err != nil {
		return fmt.Errorf("xml: decode node %q: %w", attrs.string("Name"), err)
	}
	if container, ok := node.(*connection.ContainerInfo); ok {
		container.SetExpanded(attrs.boolean("Expanded"))
	}
	if err := parent.AddChild(node); err != nil {
		return fmt.Errorf("xml: add node %q: %w", attrs.string("Name"), err)
	}
	if container, ok := node.(*connection.ContainerInfo); ok {
		for i := range element.Children {
			if err := d.addNode(container, element.Children[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d nodeDecoder) decodeNode(info *connection.ConnectionInfo, attrs attributes) error {
	if err := d.decode26(info, attrs); err != nil {
		return err
	}
	if d.version >= 27 {
		d.decode27(info, attrs)
	}
	if d.version >= 28 {
		d.decode28(info, attrs)
	}
	return nil
}

func (d nodeDecoder) decryptAttribute(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	plaintext, err := d.provider.Decrypt(compactCiphertext(value), d.password)
	if err != nil {
		return "", fmt.Errorf("%w: encrypted node attribute: %v", ErrAuthentication, err)
	}
	return plaintext, nil
}

func isIgnoredNodeType(value string) bool {
	switch strings.ToLower(value) {
	case "root", "puttyroot", "puttysession", "none":
		return true
	default:
		return false
	}
}
