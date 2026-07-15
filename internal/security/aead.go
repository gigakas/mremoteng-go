package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// AEAD is the modern mRemoteNG cryptography provider: AES-256-GCM with a
// PBKDF2-HMAC-SHA1 derived key. New connection files are written with this
// provider.
//
// The ciphertext layout matches AeadCryptographyProvider.SimpleEncrypt exactly:
//
//	base64( salt[16] || nonce[16] || ciphertext || tag[16] )
//
// The salt doubles as the GCM associated data: it is authenticated but not
// encrypted, so a tampered salt or ciphertext fails authentication. The nonce
// is 16 bytes — a non-standard GCM nonce size. cipher.NewGCM rejects non-12
// byte nonces, so cipher.NewGCMWithNonceSize is used instead; that path derives
// the initial counter from the nonce via GHASH (NIST SP 800-38D §8.1), the same
// construction BouncyCastle's GcmBlockCipher applies to the 16-byte nonce.
type AEAD struct{}

// NewAEAD returns the modern AEAD provider.
func NewAEAD() *AEAD { return &AEAD{} }

const (
	aeadSaltBytes  = 16 // SaltBitSize  = 128
	aeadNonceBytes = 16 // NonceBitSize = 128
)

// Encrypt encrypts plaintext under password, returning the Base64 ciphertext.
func (AEAD) Encrypt(plaintext string, password []byte) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}
	if plaintext == "" {
		return "", nil
	}
	salt := make([]byte, aeadSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: generate salt: %w", err)
	}
	nonce := make([]byte, aeadNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("security: generate nonce: %w", err)
	}
	gcm, err := newAEADCipher(password, salt)
	if err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), salt)
	blob := make([]byte, 0, len(salt)+len(nonce)+len(sealed))
	blob = append(blob, salt...)
	blob = append(blob, nonce...)
	blob = append(blob, sealed...)
	return base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt reverses Encrypt. It returns an error if password is wrong or the
// ciphertext has been tampered with.
func (AEAD) Decrypt(ciphertext string, password []byte) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}
	if ciphertext == "" {
		return "", nil
	}
	blob, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("security: decode ciphertext: %w", err)
	}
	if len(blob) < aeadSaltBytes+aeadNonceBytes {
		return "", fmt.Errorf("security: aead ciphertext too short (%d bytes)", len(blob))
	}
	salt := blob[:aeadSaltBytes]
	nonce := blob[aeadSaltBytes : aeadSaltBytes+aeadNonceBytes]
	sealed := blob[aeadSaltBytes+aeadNonceBytes:]
	gcm, err := newAEADCipher(password, salt)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, sealed, salt)
	if err != nil {
		return "", fmt.Errorf("security: aead decrypt: %w", err)
	}
	return string(plaintext), nil
}

// newAEADCipher derives the key from password and salt and returns the GCM
// AEAD configured for a 16-byte nonce.
func newAEADCipher(password, salt []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(deriveAEADKey(password, salt))
	if err != nil {
		return nil, fmt.Errorf("security: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, aeadNonceBytes)
	if err != nil {
		return nil, fmt.Errorf("security: gcm: %w", err)
	}
	return gcm, nil
}
