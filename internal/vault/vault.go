// Package vault stores connection secrets encrypted with a master password.
//
// The file holds a PBKDF2-HMAC-SHA256 derived key (random salt, 210k
// iterations) and an AES-256-GCM ciphertext, so a copied file cannot be read
// without the password and any modification is detected by GCM. Nothing is
// written in plaintext and no secret is ever logged.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

// File format constants.
const (
	version    = 1
	iterations = 210_000
	keyLength  = 32
	algorithm  = "pbkdf2-sha256+aes-256-gcm"
)

// ErrWrongPassword means decryption failed: a wrong password or a tampered file.
var ErrWrongPassword = errors.New("vault: wrong master password or damaged file")

// ErrLocked means the vault has no key in memory.
var ErrLocked = errors.New("vault: locked")

// Vault is an unlocked secrets store.
type Vault struct {
	path        string
	key         []byte
	aead        cipher.AEAD
	data        map[string]string
	currentSalt []byte
}

type fileFormat struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Payload    string `json:"payload"`
}

// DefaultPath returns the vault location next to the profile store.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("vault: locate config directory: %w", err)
	}
	return filepath.Join(dir, "mremoteng-go", "secrets.json"), nil
}

// Create initialises a new empty vault protected by password.
func Create(path, password string) (*Vault, error) {
	if len(password) < 8 {
		return nil, errors.New("vault: the master password needs at least 8 characters")
	}
	vault := &Vault{path: path, data: map[string]string{}}
	if err := vault.derive(password, randomBytes(16)); err != nil {
		return nil, err
	}
	return vault, nil
}

// Open unlocks an existing vault.
func Open(path, password string) (*Vault, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("vault: read: %w", err)
	}
	var file fileFormat
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("vault: decode: %w", err)
	}
	if file.Version != version || file.Algorithm != algorithm {
		return nil, fmt.Errorf("vault: unsupported format (version %d, %s)", file.Version, file.Algorithm)
	}
	salt, err := base64.StdEncoding.DecodeString(file.Salt)
	if err != nil {
		return nil, fmt.Errorf("vault: salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(file.Nonce)
	if err != nil {
		return nil, fmt.Errorf("vault: nonce: %w", err)
	}
	payload, err := base64.StdEncoding.DecodeString(file.Payload)
	if err != nil {
		return nil, fmt.Errorf("vault: payload: %w", err)
	}
	vault := &Vault{path: path}
	if err := vault.derive(password, salt); err != nil {
		return nil, err
	}
	plaintext, err := vault.aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, ErrWrongPassword
	}
	vault.data = map[string]string{}
	if len(plaintext) > 0 {
		if err := json.Unmarshal(plaintext, &vault.data); err != nil {
			return nil, fmt.Errorf("vault: decode secrets: %w", err)
		}
	}
	return vault, nil
}

// Exists reports whether a vault file is present.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// derive builds the key and cipher for password.
func (v *Vault) derive(password string, salt []byte) error {
	v.currentSalt = append([]byte(nil), salt...)
	v.key = pbkdf2.Key([]byte(password), salt, iterations, keyLength, sha256.New)
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return fmt.Errorf("vault: cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("vault: gcm: %w", err)
	}
	v.aead = aead
	return nil
}

// Get returns the stored secret for id.
func (v *Vault) Get(id string) (string, bool) {
	if v == nil || v.aead == nil {
		return "", false
	}
	secret, ok := v.data[id]
	return secret, ok
}

// Set stores or replaces the secret for id.
func (v *Vault) Set(id, secret string) error {
	if v == nil || v.aead == nil {
		return ErrLocked
	}
	if id == "" {
		return errors.New("vault: missing connection id")
	}
	v.data[id] = secret
	return nil
}

// Delete removes the secret for id.
func (v *Vault) Delete(id string) {
	if v == nil || v.aead == nil || id == "" {
		return
	}
	delete(v.data, id)
}

// Save writes the encrypted vault atomically with owner-only permissions.
func (v *Vault) Save() error {
	if v == nil || v.aead == nil {
		return ErrLocked
	}
	plaintext, err := json.Marshal(v.data)
	if err != nil {
		return fmt.Errorf("vault: encode secrets: %w", err)
	}
	nonce := randomBytes(v.aead.NonceSize())
	ciphertext := v.aead.Seal(nil, nonce, plaintext, nil)
	// The salt lives in the file; re-derive needs it, so keep it alongside.
	salt := v.salt()
	file := fileFormat{
		Version:    version,
		Algorithm:  algorithm,
		Iterations: iterations,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Payload:    base64.StdEncoding.EncodeToString(ciphertext),
	}
	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("vault: encode file: %w", err)
	}
	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("vault: create %s: %w", dir, err)
	}
	temp, err := os.CreateTemp(dir, ".secrets-*.json")
	if err != nil {
		return fmt.Errorf("vault: temporary file: %w", err)
	}
	name := temp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("vault: permissions: %w", err)
	}
	if _, err := temp.Write(append(encoded, '\n')); err != nil {
		_ = temp.Close()
		return fmt.Errorf("vault: write: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("vault: sync: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("vault: close: %w", err)
	}
	if err := os.Rename(name, v.path); err != nil {
		return fmt.Errorf("vault: replace: %w", err)
	}
	return nil
}

// Lock forgets the key and the secrets.
func (v *Vault) Lock() {
	if v == nil {
		return
	}
	v.key = nil
	v.aead = nil
	v.data = nil
}

// Locked reports whether the vault lost its key.
func (v *Vault) Locked() bool { return v == nil || v.aead == nil }

func randomBytes(size int) []byte {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		panic("vault: entropy source failed: " + err.Error())
	}
	return buffer
}

// salt returns the salt used for the current key. It is persisted in Save.
func (v *Vault) salt() []byte { return v.currentSalt }
