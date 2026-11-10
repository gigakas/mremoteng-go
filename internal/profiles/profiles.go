// Package profiles owns the application's native connection profiles: the
// protocol-neutral record, its validation rules and the local JSON store.
//
// The native store is deliberately independent of the mRemoteNG XML format.
// It holds endpoint metadata only; passwords are never persisted here.
package profiles

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Protocol identifies how a profile is reached.
type Protocol string

const (
	// SSH opens a native SSH session (implemented in the session stage).
	SSH Protocol = "SSH"
	// RDP launches an external FreeRDP process (implemented in the session stage).
	RDP Protocol = "RDP"
	// VNC uses the native RFB client.
	VNC Protocol = "VNC"
	// TELNET, RAW and RLOGIN use the byte-stream clients and the terminal view.
	TELNET Protocol = "TELNET"
	RAW    Protocol = "RAW"
	RLOGIN Protocol = "RLOGIN"
	// HTTP opens the address in the system browser.
	HTTP Protocol = "HTTP"
)

var (
	ErrMissingName      = errors.New("profile: name is required")
	ErrInvalidName      = errors.New("profile: name contains invalid characters")
	ErrInvalidProtocol  = errors.New("profile: protocol must be SSH or RDP")
	ErrMissingHost      = errors.New("profile: host is required")
	ErrInvalidHost      = errors.New("profile: host contains invalid characters")
	ErrInvalidPort      = errors.New("profile: port must be between 1 and 65535")
	ErrInvalidUsername  = errors.New("profile: username contains invalid characters")
	ErrInvalidFolder    = errors.New("profile: folder path is invalid")
	ErrInvalidID        = errors.New("profile: ID must not be empty")
	ErrDuplicateID      = errors.New("profile: duplicate ID")
	ErrUnsupportedStore = errors.New("profile store: unsupported schema version")
	ErrInvalidStore     = errors.New("profile store: invalid data")
)

const (
	maxNameLength     = 128
	maxHostLength     = 253
	maxUsernameLength = 128
	maxFolderLength   = 256
	maxFolderSegment  = 64
)

// Profile is one saved connection target.
type Profile struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol Protocol `json:"protocol"`
	Folder   string   `json:"folder,omitempty"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username,omitempty"`

	// Folder-level inheritance. When enabled, the value resolves from the
	// closest ancestor folder that defines it, so a team folder can hold the
	// shared username or port.
	InheritUsername bool `json:"inherit_username,omitempty"`
	InheritPort     bool `json:"inherit_port,omitempty"`
}

// FolderDefault holds the values a folder shares with its descendants.
type FolderDefault struct {
	Path     string `json:"path"`
	Username string `json:"username,omitempty"`
	Port     int    `json:"port,omitempty"`
}

// FolderSegments splits a folder path into its segments. An empty folder means
// the tree root. Segments are separated by "/" and must not be empty, "." or
// ".." so the tree cannot escape or alias a level.
func FolderSegments(folder string) ([]string, error) {
	if folder == "" {
		return nil, nil
	}
	if len(folder) > maxFolderLength || hasControlRunes(folder) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidFolder, folder)
	}
	segments := strings.Split(folder, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || len(segment) > maxFolderSegment {
			return nil, fmt.Errorf("%w: %q", ErrInvalidFolder, folder)
		}
	}
	return segments, nil
}

// NewID returns a random RFC 4122 version 4 identifier for a new profile.
func NewID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("profile: generate ID: %w", err)
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}

// DefaultPort returns the conventional port for protocol.
func DefaultPort(protocol Protocol) int {
	switch protocol {
	case SSH:
		return 22
	case RDP:
		return 3389
	case VNC:
		return 5900
	case TELNET, RAW:
		return 23
	case RLOGIN:
		return 513
	case HTTP:
		return 80
	default:
		return 0
	}
}

// Validate reports whether the profile can be saved. It rejects values that
// would be ambiguous in the store or dangerous when passed to an external
// client process, such as fields starting with "-".
func (p Profile) Validate() error {
	switch {
	case strings.TrimSpace(p.Name) == "":
		return ErrMissingName
	case len(p.Name) > maxNameLength || hasControlRunes(p.Name):
		return fmt.Errorf("%w: %q", ErrInvalidName, p.Name)
	}
	switch p.Protocol {
	case SSH, RDP, VNC, TELNET, RAW, RLOGIN, HTTP:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidProtocol, p.Protocol)
	}
	if _, err := FolderSegments(p.Folder); err != nil {
		return err
	}
	host := strings.TrimSpace(p.Host)
	switch {
	case host == "":
		return ErrMissingHost
	case len(host) > maxHostLength, strings.HasPrefix(host, "-"), hasControlRunes(host), hasSpaceRunes(host),
		strings.Contains(host, "://"), strings.Contains(host, "/"), strings.Contains(host, "@"):
		return fmt.Errorf("%w: %q", ErrInvalidHost, p.Host)
	case strings.Contains(host, ":") && !strings.HasPrefix(host, "["):
		// A bare colon is ambiguous (host:port) or an unbracketed IPv6
		// literal; require the bracketed form instead of guessing.
		return fmt.Errorf("%w: %q", ErrInvalidHost, p.Host)
	}
	if !p.InheritPort && (p.Port < 1 || p.Port > 65535) {
		return fmt.Errorf("%w: %d", ErrInvalidPort, p.Port)
	}
	if p.InheritPort && p.Port != 0 && (p.Port < 1 || p.Port > 65535) {
		return fmt.Errorf("%w: %d", ErrInvalidPort, p.Port)
	}
	if p.Username != "" {
		if len(p.Username) > maxUsernameLength || strings.HasPrefix(p.Username, "-") || hasControlRunes(p.Username) || hasSpaceRunes(p.Username) {
			return fmt.Errorf("%w: %q", ErrInvalidUsername, p.Username)
		}
	}
	return nil
}

func hasControlRunes(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func hasSpaceRunes(value string) bool {
	return strings.ContainsFunc(value, unicode.IsSpace)
}

// Resolve returns the profile with inherited values applied. The closest
// ancestor folder wins per field; when it does not define a field the search
// continues upwards.
func Resolve(profile Profile, defaults []FolderDefault) Profile {
	resolved := profile
	byPath := make(map[string]FolderDefault, len(defaults))
	for _, item := range defaults {
		byPath[item.Path] = item
	}
	needUsername, needPort := profile.InheritUsername, profile.InheritPort
	for folder := profile.Folder; folder != "" && (needUsername || needPort); {
		if item, ok := byPath[folder]; ok {
			if needUsername && item.Username != "" {
				resolved.Username = item.Username
				needUsername = false
			}
			if needPort && item.Port != 0 {
				resolved.Port = item.Port
				needPort = false
			}
		}
		index := strings.LastIndex(folder, "/")
		if index < 0 {
			break
		}
		folder = folder[:index]
	}
	return resolved
}
