package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// schemaVersion is the native store version written by this build. Version 1
// lacked folder defaults and is migrated on load.
const schemaVersion = 2

// Document is the complete native store content: the connection profiles and
// the folders the user created (which may still be empty).
type Document struct {
	Profiles []Profile
	Folders  []string
	Defaults []FolderDefault
	// Collapsed lists folder paths the user collapsed in the tree, so the
	// workspace looks the same after a restart.
	Collapsed []string
}

// Store is the local JSON-backed profile store. It is the native application
// storage; importing mRemoteNG files is a separate, optional adapter.
type Store struct {
	path string
}

type storeFile struct {
	Version   int             `json:"version"`
	Folders   []string        `json:"folders,omitempty"`
	Defaults  []FolderDefault `json:"folder_defaults,omitempty"`
	Collapsed []string        `json:"collapsed_folders,omitempty"`
	Profiles  []Profile       `json:"profiles"`
}

// DefaultPath returns the native store location, honouring XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("profile store: locate config directory: %w", err)
	}
	return filepath.Join(dir, "mremoteng-go", "profiles.json"), nil
}

// NewStore returns a store reading and writing path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Path reports the file backing the store.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Load returns the stored profiles, or an empty slice when the file does not
// exist yet. Corrupt or unsupported data is reported as an error so callers
// never overwrite a file they could not read.
func (s *Store) Load() (Document, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Document{Profiles: []Profile{}}, nil
	}
	if err != nil {
		return Document{}, fmt.Errorf("profile store: read %s: %w", s.path, err)
	}
	var file storeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return Document{}, fmt.Errorf("profile store: decode %s: %w", s.path, err)
	}
	switch file.Version {
	case 1:
		// Version 1 has no folder defaults; it is read as-is and rewritten as 2.
	case schemaVersion:
	default:
		return Document{}, fmt.Errorf("%w: %d in %s", ErrUnsupportedStore, file.Version, s.path)
	}
	if err := validateProfiles(file.Profiles); err != nil {
		return Document{}, fmt.Errorf("%w: %s: %v", ErrInvalidStore, s.path, err)
	}
	if err := validateFolders(file.Folders); err != nil {
		return Document{}, fmt.Errorf("%w: %s: %v", ErrInvalidStore, s.path, err)
	}
	if err := validateDefaults(file.Defaults); err != nil {
		return Document{}, fmt.Errorf("%w: %s: %v", ErrInvalidStore, s.path, err)
	}
	if err := validateFolders(file.Collapsed); err != nil {
		return Document{}, fmt.Errorf("%w: %s: %v", ErrInvalidStore, s.path, err)
	}
	document := Document{Profiles: file.Profiles, Folders: file.Folders, Defaults: file.Defaults, Collapsed: file.Collapsed}
	if document.Profiles == nil {
		document.Profiles = []Profile{}
	}
	return document, nil
}

// Save atomically replaces the store with profiles. The temporary file is
// created in the target directory with owner-only permissions so a failed
// write never truncates existing data.
func (s *Store) Save(document Document) error {
	if err := validateProfiles(document.Profiles); err != nil {
		return err
	}
	if err := validateFolders(document.Folders); err != nil {
		return err
	}
	if err := validateDefaults(document.Defaults); err != nil {
		return err
	}
	if err := validateFolders(document.Collapsed); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(storeFile{
		Version:   schemaVersion,
		Folders:   document.Folders,
		Defaults:  document.Defaults,
		Collapsed: document.Collapsed,
		Profiles:  document.Profiles,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("profile store: encode: %w", err)
	}
	payload = append(payload, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("profile store: create %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".profiles-*.json")
	if err != nil {
		return fmt.Errorf("profile store: create temporary file: %w", err)
	}
	tempName := temp.Name()
	defer func() {
		// Cleanup after a failed write; the successful path renamed the file.
		// A failure here must not mask the original write error.
		_ = os.Remove(tempName)
	}()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("profile store: restrict permissions: %w", err)
	}
	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		return fmt.Errorf("profile store: write temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("profile store: sync temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("profile store: close temporary file: %w", err)
	}
	if err := os.Rename(tempName, s.path); err != nil {
		return fmt.Errorf("profile store: replace %s: %w", s.path, err)
	}
	return nil
}

// validateFolders rejects malformed or duplicated folder paths.
func validateFolders(folders []string) error {
	seen := make(map[string]bool, len(folders))
	for _, folder := range folders {
		segments, err := FolderSegments(folder)
		if err != nil {
			return err
		}
		if len(segments) == 0 {
			return fmt.Errorf("%w: empty folder path", ErrInvalidFolder)
		}
		if seen[folder] {
			return fmt.Errorf("%w: %s", ErrDuplicateID, folder)
		}
		seen[folder] = true
	}
	return nil
}

// validateDefaults rejects malformed folder default records.
func validateDefaults(defaults []FolderDefault) error {
	seen := make(map[string]bool, len(defaults))
	for _, item := range defaults {
		segments, err := FolderSegments(item.Path)
		if err != nil || len(segments) == 0 {
			return fmt.Errorf("%w: %q", ErrInvalidFolder, item.Path)
		}
		if seen[item.Path] {
			return fmt.Errorf("%w: folder default %s", ErrDuplicateID, item.Path)
		}
		seen[item.Path] = true
		if item.Port != 0 && (item.Port < 1 || item.Port > 65535) {
			return fmt.Errorf("%w: %s: %d", ErrInvalidPort, item.Path, item.Port)
		}
		if item.Username != "" && (len(item.Username) > maxUsernameLength || strings.HasPrefix(item.Username, "-") ||
			hasControlRunes(item.Username) || hasSpaceRunes(item.Username)) {
			return fmt.Errorf("%w: %s: %q", ErrInvalidUsername, item.Path, item.Username)
		}
	}
	return nil
}

func validateProfiles(profiles []Profile) error {
	seen := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		if profile.ID == "" {
			return fmt.Errorf("%w: %q", ErrInvalidID, profile.Name)
		}
		if seen[profile.ID] {
			return fmt.Errorf("%w: %s", ErrDuplicateID, profile.ID)
		}
		seen[profile.ID] = true
		if err := profile.Validate(); err != nil {
			return err
		}
	}
	return nil
}
