// Package security implements the symmetric encryption schemes used by the
// original mRemoteNG to protect connection files.
//
// There are two providers, mirroring the C# cryptography classes named in the
// migration plan:
//
//   - AEAD: the modern AES-256-GCM provider with PBKDF2-HMAC-SHA1 key
//     derivation (AeadCryptographyProvider + Pkcs5S2KeyGenerator). This is the
//     format written by new files.
//   - Legacy: the original AES-128-CBC provider with an MD5-derived key
//     (LegacyRijndaelCryptographyProvider). Retained to read older files.
//
// Both providers exchange the exact Base64 strings the C# application stores
// in connection files: a value encrypted by mRemoteNG decrypts here and
// vice-versa. The on-the-wire layout and parameters match the C# sources
// byte-for-byte; see aead.go and legacy.go for the per-format detail.
package security

import "errors"

// Provider abstracts a symmetric encryption scheme that encrypts and decrypts
// the Base64 strings stored in connection files. password carries the UTF-8
// bytes of the file password; each provider encodes it the way the matching
// C# provider does.
type Provider interface {
	// Encrypt returns the ciphertext of plaintext as a Base64 string,
	// laid out as mRemoteNG stores it.
	Encrypt(plaintext string, password []byte) (string, error)
	// Decrypt reverses Encrypt. It returns an error if the password is
	// wrong or the ciphertext is tampered with (for authenticated
	// providers); the legacy CBC provider has no authentication, so a wrong
	// password may instead yield a padding error or garbage.
	Decrypt(ciphertext string, password []byte) (string, error)
}

// ErrEmptyPassword is returned when a provider is asked to encrypt or decrypt
// with an empty password.
var ErrEmptyPassword = errors.New("security: empty password")

// ErrInvalidIterations is returned when PBKDF2 is configured below the
// minimum accepted by mRemoteNG's Pkcs5S2KeyGenerator.
var ErrInvalidIterations = errors.New("security: PBKDF2 iterations must be at least 1000")

// compile-time interface checks.
var (
	_ Provider = (*AEAD)(nil)
	_ Provider = (*Legacy)(nil)
)
