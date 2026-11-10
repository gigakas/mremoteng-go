// Package settings persists application-level preferences (window size,
// the last-opened connections file, theme choice) to a plain JSON file —
// not the Windows registry the original C# app used
// (Config/Settings/Registry/), per the blueprint's explicit v1 decision
// ("settings in a plain config file (no Windows registry)").
//
// Enterprise-deployment equivalent of the original registry policies:
// the original app supports machine-wide policy via registry keys under
// HKLM, read at startup to lock down or default certain settings for
// managed deployments. This package has no equivalent yet — a v2
// addition could check an OS-appropriate machine-wide config file
// (e.g. /etc/mremoteng/settings.json on Linux, a Program Files-relative
// path on Windows) before falling back to the per-user file Load reads,
// letting administrators ship a template the way a GPO-deployed registry
// key would have. Not implemented here since there is no consumer for it
// yet (no fleet deployment story exists until much later phases).
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// fileName is the settings file's name within its containing directory
// (see DefaultPath).
const fileName = "settings.json"

// Settings holds the application-level preferences this v1 persists.
// Connection *data* (the tree itself) is a separate concern, saved to
// whatever file the user opened/saved via internal/serialize/xml — this
// struct only remembers *which* file that was, plus window geometry and
// (for stage 3.6, once it exists) the chosen theme.
type Settings struct {
	WindowWidth         float32 `json:"windowWidth"`
	WindowHeight        float32 `json:"windowHeight"`
	LastConnectionsFile string  `json:"lastConnectionsFile"`
	Theme               string  `json:"theme"`
}

// Default returns the settings a fresh install starts with.
func Default() *Settings {
	return &Settings{
		WindowWidth:  1024,
		WindowHeight: 768,
		Theme:        "system",
	}
}

// DefaultPath returns the per-user path Load/Save use when called with an
// empty path: os.UserConfigDir()/mremoteng-go/settings.json.
// os.UserConfigDir() already resolves to the correct OS convention
// (%AppData% on Windows, $XDG_CONFIG_HOME or ~/.config on Linux) — no
// platform-specific code needed here.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("settings: resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "mremoteng-go", fileName), nil
}

// Load reads settings from path (DefaultPath() if path is empty). A
// missing file is not an error — it returns Default(), the expected
// state on first run.
func Load(path string) (*Settings, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: read %s: %w", path, err)
	}

	s := Default()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("settings: parse %s: %w", path, err)
	}
	return s, nil
}

// Save writes s to path (DefaultPath() if path is empty), creating the
// containing directory if needed.
func (s *Settings) Save(path string) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("settings: create directory for %s: %w", path, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("settings: encode: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("settings: write %s: %w", path, err)
	}
	return nil
}
