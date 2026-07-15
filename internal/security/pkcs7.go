package security

import "errors"

// pkcs7Pad applies PKCS#7 padding so data is a whole multiple of blockSize
// bytes. The C# providers use PKCS7 (the default for System.Security.Cryptography
// symmetric ciphers), so the padding here must match byte-for-byte.
func pkcs7Pad(data []byte, blockSize int) []byte {
	n := blockSize - len(data)%blockSize
	pad := make([]byte, n)
	for i := range pad {
		pad[i] = byte(n)
	}
	return append(data, pad...)
}

// pkcs7Unpad reverses pkcs7Pad and rejects malformed padding.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 0 {
		return nil, errors.New("invalid block size")
	}
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid padding: data is not block-aligned")
	}
	n := int(data[len(data)-1])
	if n == 0 || n > blockSize || n > len(data) {
		return nil, errors.New("invalid padding: bad pad length")
	}
	for i := len(data) - n; i < len(data); i++ {
		if data[i] != byte(n) {
			return nil, errors.New("invalid padding: inconsistent bytes")
		}
	}
	return data[:len(data)-n], nil
}
