package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Legacy is the original mRemoteNG cryptography provider: AES-128-CBC with
// PKCS#7 padding, the key derived as MD5 of the UTF-8 password
// (LegacyRijndaelCryptographyProvider). New files must not use it — prefer
// AEAD; it is retained to read connection files written by older releases.
//
// The ciphertext layout matches the C# provider:
//
//	base64( iv[16] || ciphertext )
//
// There is no authentication: CBC alone cannot detect a wrong password or
// tampering, so decrypting with the wrong password either fails with a padding
// error or returns garbage plaintext.
type Legacy struct{}

// NewLegacy returns the legacy AES-128-CBC provider.
func NewLegacy() *Legacy { return &Legacy{} }

// Encrypt encrypts plaintext under password, returning the Base64 ciphertext.
func (Legacy) Encrypt(plaintext string, password []byte) (string, error) {
	if len(password) == 0 {
		return "", ErrEmptyPassword
	}
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(legacyKey(password))
	if err != nil {
		return "", fmt.Errorf("security: aes cipher: %w", err)
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("security: generate iv: %w", err)
	}
	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	blob := make([]byte, 0, len(iv)+len(ciphertext))
	blob = append(blob, iv...)
	blob = append(blob, ciphertext...)
	return base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt reverses Encrypt. A wrong password yields a padding error or garbage
// rather than a clean failure.
func (Legacy) Decrypt(ciphertext string, password []byte) (string, error) {
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
	if len(blob) < aes.BlockSize {
		return "", fmt.Errorf("security: legacy ciphertext too short (%d bytes)", len(blob))
	}
	iv := blob[:aes.BlockSize]
	body := blob[aes.BlockSize:]
	if len(body)%aes.BlockSize != 0 {
		return "", fmt.Errorf("security: legacy ciphertext length %d is not a multiple of the block size", len(body))
	}
	block, err := aes.NewCipher(legacyKey(password))
	if err != nil {
		return "", fmt.Errorf("security: aes cipher: %w", err)
	}
	padded := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(padded, body)
	plaintext, err := pkcs7Unpad(padded, aes.BlockSize)
	if err != nil {
		return "", fmt.Errorf("security: legacy decrypt: %w", err)
	}
	return string(plaintext), nil
}

// legacyKey derives the 128-bit AES key as the original provider does:
// MD5 of the UTF-8 password bytes.
func legacyKey(password []byte) []byte {
	sum := md5.Sum(password)
	return sum[:]
}
