// Package transfer maps mRemoteNG connection files (XML and CSV) to and from the
// native profile model. It never writes to the imported file and never carries
// imported passwords into the native store.
package transfer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mRemoteNG/mremoteng-go/internal/connection"
	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	"github.com/mRemoteNG/mremoteng-go/internal/serialize/csv"
	connectionxml "github.com/mRemoteNG/mremoteng-go/internal/serialize/xml"
)

// Format identifies a supported file format.
type Format string

const (
	FormatXML Format = "XML"
	FormatCSV Format = "CSV"
)

// Preview is what an import would add, before anything is written.
type Preview struct {
	Format      Format
	Profiles    []profiles.Profile
	Folders     []string
	Unsupported int
	Dropped     int
	// Encrypted reports that the source XML declared protection.
	Encrypted bool
}

// ErrNeedPassword means the file is protected and no usable password was given.
var ErrNeedPassword = errors.New("transfer: the file is protected; a password is required")

// ReadXML parses an mRemoteNG connection file.
func ReadXML(data []byte, password string) (Preview, error) {
	document, err := connectionxml.Deserialize(data, connectionxml.Options{Password: []byte(password)})
	if err != nil {
		if errors.Is(err, connectionxml.ErrAuthentication) {
			if password == "" {
				return Preview{Format: FormatXML}, ErrNeedPassword
			}
			return Preview{Format: FormatXML, Encrypted: true}, fmt.Errorf("transfer: decrypt: %w", err)
		}
		return Preview{Format: FormatXML}, err
	}
	preview := Preview{Format: FormatXML, Encrypted: document.Metadata.Protected != "" && document.Metadata.Protected != "ThisIsNotProtected"}
	preview.collect(document.Root, "")
	return preview, nil
}

// ReadCSV parses an mRemoteNG CSV export.
func ReadCSV(data []byte) (Preview, error) {
	root, err := csv.Deserialize(data)
	if err != nil {
		return Preview{Format: FormatCSV}, err
	}
	preview := Preview{Format: FormatCSV}
	preview.collect(root, "")
	return preview, nil
}

// collect walks a connection tree, turning containers into folder paths.
func (p *Preview) collect(node connection.Node, parent string) {
	base := node.Base()
	values := base.Raw
	switch node.Kind() {
	case connection.NodeKindRoot:
		for _, child := range children(node) {
			p.collect(child, parent)
		}
		return
	case connection.NodeKindContainer:
		folder := joinFolder(parent, sanitizeSegment(values.Name))
		if folder != "" {
			p.Folders = append(p.Folders, folder)
		}
		for _, child := range children(node) {
			p.collect(child, folder)
		}
		return
	}

	protocol, supported := nativeProtocol(values.Protocol)
	if !supported {
		p.Unsupported++
		return
	}
	if values.Password != "" {
		p.Dropped++
	}
	id, err := profiles.NewID()
	if err != nil {
		p.Unsupported++
		return
	}
	port := values.Port
	if port == 0 {
		port = profiles.DefaultPort(protocol)
	}
	profile := profiles.Profile{
		ID:       id,
		Name:     strings.TrimSpace(values.Name),
		Folder:   parent,
		Protocol: protocol,
		Host:     strings.TrimSpace(values.Hostname),
		Port:     port,
		Username: strings.TrimSpace(values.Username),
	}
	if profile.Name == "" {
		profile.Name = "Imported connection"
	}
	if err := profile.Validate(); err != nil {
		// Unusable rows are reported instead of silently stored broken.
		p.Unsupported++
		return
	}
	p.Profiles = append(p.Profiles, profile)
}

// children returns the direct children of a node.
func children(node connection.Node) []connection.Node {
	container, ok := node.(*connection.ContainerInfo)
	if !ok {
		return nil
	}
	return container.Children()
}

// joinFolder appends a sanitised segment to a folder path.
func joinFolder(parent, segment string) string {
	if segment == "" {
		return parent
	}
	if parent == "" {
		return segment
	}
	return parent + "/" + segment
}

// sanitizeSegment makes a container name usable as a folder path segment.
func sanitizeSegment(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "-")
	replacer := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")
	name = replacer.Replace(name)
	if _, err := profiles.FolderSegments(name); err != nil {
		return "Imported folder"
	}
	return name
}

// nativeProtocol maps mRemoteNG protocol names to the two protocols the
// application can actually use.
func nativeProtocol(protocol connection.ProtocolType) (profiles.Protocol, bool) {
	switch protocol {
	case connection.ProtocolSSH1, connection.ProtocolSSH2, connection.ProtocolTerminal:
		return profiles.SSH, true
	case connection.ProtocolRDP:
		return profiles.RDP, true
	default:
		return "", false
	}
}

// WriteXML exports profiles and folders as a v2.8 connection file. An empty
// password produces an unprotected file.
func WriteXML(all []profiles.Profile, folders []string, password string) ([]byte, error) {
	root, err := buildTree(all, folders)
	if err != nil {
		return nil, err
	}
	document := &connectionxml.Document{Root: root, Metadata: connectionxml.Metadata{Name: "Connections", ConfVersion: "2.8"}}
	return connectionxml.Serialize(document, connectionxml.SerializeOptions{Password: []byte(password)})
}

// WriteCSV exports profiles and folders in the mRemoteNG CSV layout.
func WriteCSV(all []profiles.Profile, folders []string) ([]byte, error) {
	root, err := buildTree(all, folders)
	if err != nil {
		return nil, err
	}
	return csv.Serialize(root, csv.SerializeOptions{})
}

// buildTree recreates a connection tree from native profiles and folders.
func buildTree(all []profiles.Profile, folders []string) (*connection.ContainerInfo, error) {
	root, err := connection.NewRootInfo()
	if err != nil {
		return nil, err
	}
	containers := map[string]*connection.ContainerInfo{"": root}
	for _, folder := range folders {
		if _, err := ensureContainer(root, containers, folder); err != nil {
			return nil, err
		}
	}
	for _, profile := range all {
		parent, err := ensureContainer(root, containers, profile.Folder)
		if err != nil {
			return nil, err
		}
		info, err := connection.NewConnectionInfoWithID(profile.ID)
		if err != nil {
			return nil, err
		}
		info.Raw.Name = profile.Name
		info.Raw.Hostname = profile.Host
		info.Raw.Port = profile.Port
		info.Raw.Username = profile.Username
		info.Raw.Protocol = treeProtocol(profile.Protocol)
		if err := parent.AddChild(info); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func treeProtocol(protocol profiles.Protocol) connection.ProtocolType {
	if protocol == profiles.RDP {
		return connection.ProtocolRDP
	}
	return connection.ProtocolSSH2
}

// ensureContainer returns the container for a folder path, creating it and its
// ancestors when needed.
func ensureContainer(root *connection.ContainerInfo, known map[string]*connection.ContainerInfo, folder string) (*connection.ContainerInfo, error) {
	if folder == "" {
		return root, nil
	}
	if container, ok := known[folder]; ok {
		return container, nil
	}
	segments, err := profiles.FolderSegments(folder)
	if err != nil {
		return nil, err
	}
	parent := root
	path := ""
	for _, segment := range segments {
		if path == "" {
			path = segment
		} else {
			path += "/" + segment
		}
		if container, ok := known[path]; ok {
			parent = container
			continue
		}
		container, err := connection.NewContainerInfo()
		if err != nil {
			return nil, err
		}
		container.Base().Raw.Name = segment
		if err := parent.AddChild(container); err != nil {
			return nil, err
		}
		known[path] = container
		parent = container
	}
	return parent, nil
}
