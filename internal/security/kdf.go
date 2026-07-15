package security

import (
	"crypto/sha1"
	"unicode/utf16"

	"golang.org/x/crypto/pbkdf2"
)

// mRemoteNG AEAD key-derivation and layout parameters. The original project
// derives these from BouncyCastle's Pkcs5S2ParametersGenerator (PBKDF2 over
// HMAC-SHA1, the default digest) with 1000 iterations and a 256-bit key. SHA-1
// and 1000 iterations are weak by modern standards but are part of the file
// format: changing them would break interoperability with existing files, so
// they are kept here for format parity. A v2 format bump would be the place to
// strengthen them.
const (
	pbkdf2Iterations = 1000
	aeadKeyBytes     = 32 // KeyBitSize = 256
)

// deriveAEADKey runs the same PBKDF2-HMAC-SHA1 mRemoteNG uses to turn the file
// password and the per-message salt into a 256-bit AES key. password carries
// the UTF-8 bytes of the password; it is re-encoded the way the C# provider
// does (see pkcs5PasswordToBytes).
func deriveAEADKey(password, salt []byte) []byte {
	return pbkdf2.Key(
		pkcs5PasswordToBytes(string(password)),
		salt,
		pbkdf2Iterations,
		aeadKeyBytes,
		sha1.New,
	)
}

// pkcs5PasswordToBytes mirrors BouncyCastle's
// PbeParametersGenerator.Pkcs5PasswordToBytes(char[]), which the original
// AeadCryptographyProvider applies to password.ToCharArray(). It takes the low
// 8 bits of each UTF-16 code unit, i.e. ISO-8859-1 for code points below U+0100
// and the low byte of each surrogate otherwise. For ASCII passwords this is
// identical to the raw bytes.
func pkcs5PasswordToBytes(passwordUTF8 string) []byte {
	units := utf16.Encode([]rune(passwordUTF8))
	out := make([]byte, len(units))
	for i, u := range units {
		out[i] = byte(u)
	}
	return out
}
