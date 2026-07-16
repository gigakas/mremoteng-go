package security

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestAEAD_RoundTrip_RecoversPlaintext(t *testing.T) {
	cases := []struct {
		name      string
		plaintext string
		password  string
	}{
		{"simple", "hello", "secret"},
		{"unicode", "contraseña: 日本語 ✓", "p@ssw0rd"},
		{"long", strings.Repeat("abcdefgh", 64), "longpassword"},
		{"single byte", "x", "p"},
	}
	for _, c := range cases {
		enc, err := NewAEAD().Encrypt(c.plaintext, []byte(c.password))
		if err != nil {
			t.Errorf("%s: encrypt error: %v", c.name, err)
			continue
		}
		got, err := NewAEAD().Decrypt(enc, []byte(c.password))
		if err != nil {
			t.Errorf("%s: decrypt error: %v", c.name, err)
			continue
		}
		if got != c.plaintext {
			t.Errorf("%s: got %q, want %q", c.name, got, c.plaintext)
		}
	}
}

func TestAEAD_WrongPassword_ReturnsError(t *testing.T) {
	enc, err := NewAEAD().Encrypt("secret message", []byte("right"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewAEAD().Decrypt(enc, []byte("wrong")); err == nil {
		t.Error("expected authentication error, got none")
	}
}

func TestAEAD_TamperedCiphertext_ReturnsError(t *testing.T) {
	enc, err := NewAEAD().Encrypt("secret message", []byte("right"))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	// Flip the last ciphertext byte (the tag is appended after the ciphertext).
	blob[len(blob)-1] ^= 0x01
	tampered := base64.StdEncoding.EncodeToString(blob)
	if _, err := NewAEAD().Decrypt(tampered, []byte("right")); err == nil {
		t.Error("expected authentication error for tampered ciphertext, got none")
	}
}

func TestAEAD_TamperedSalt_ReturnsError(t *testing.T) {
	enc, _ := NewAEAD().Encrypt("secret message", []byte("right"))
	blob, _ := base64.StdEncoding.DecodeString(enc)
	blob[0] ^= 0x01 // mutate a salt byte
	tampered := base64.StdEncoding.EncodeToString(blob)
	if _, err := NewAEAD().Decrypt(tampered, []byte("right")); err == nil {
		t.Error("expected authentication error for tampered salt, got none")
	}
}

func TestAEAD_EachEncryptionDiffers(t *testing.T) {
	// Random salt and nonce mean two encryptions of the same plaintext differ.
	a, _ := NewAEAD().Encrypt("same", []byte("pw"))
	b, _ := NewAEAD().Encrypt("same", []byte("pw"))
	if a == b {
		t.Error("two encryptions produced identical ciphertext; nonce/salt not random")
	}
}

func TestAEAD_CiphertextLayout(t *testing.T) {
	plaintext := "payload"
	enc, err := NewAEAD().Encrypt(plaintext, []byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	// Minimum layout: salt(16) + nonce(16) + at least one plaintext block is
	// not required (GCM is a stream cipher), but the tag is always 16 bytes.
	min := aeadSaltBytes + aeadNonceBytes + len(plaintext) + 16
	if len(blob) < min {
		t.Errorf("blob length = %d, want at least %d", len(blob), min)
	}
}

func TestAEAD_EmptyInputs(t *testing.T) {
	if got, err := NewAEAD().Encrypt("", []byte("pw")); got != "" || err != nil {
		t.Errorf(`empty plaintext: got (%q,%v), want ("",nil)`, got, err)
	}
	if got, err := NewAEAD().Decrypt("", []byte("pw")); got != "" || err != nil {
		t.Errorf(`empty ciphertext: got (%q,%v), want ("",nil)`, got, err)
	}
	if _, err := NewAEAD().Encrypt("x", nil); !errors.Is(err, ErrEmptyPassword) {
		t.Errorf("nil password: got %v, want ErrEmptyPassword", err)
	}
	if _, err := NewAEAD().Decrypt("x", nil); !errors.Is(err, ErrEmptyPassword) {
		t.Errorf("nil password: got %v, want ErrEmptyPassword", err)
	}
}

func TestAEAD_MalformedInput_ReturnsError(t *testing.T) {
	if _, err := NewAEAD().Decrypt("!!!not base64!!!", []byte("pw")); err == nil {
		t.Error("expected error for invalid base64, got none")
	}
	// Valid base64 but too short to hold salt+nonce.
	short := base64.StdEncoding.EncodeToString([]byte("tiny"))
	if _, err := NewAEAD().Decrypt(short, []byte("pw")); err == nil {
		t.Error("expected error for truncated ciphertext, got none")
	}
}

func TestAEAD_CustomIterations_RoundTripAndRejectsDefaultProvider(t *testing.T) {
	provider, err := NewAEADWithIterations(5000)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := provider.Encrypt("custom iterations", []byte("password"))
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := provider.Decrypt(ciphertext, []byte("password"))
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "custom iterations" {
		t.Errorf("plaintext = %q, want custom iterations", plaintext)
	}
	if _, err := NewAEAD().Decrypt(ciphertext, []byte("password")); err == nil {
		t.Error("default provider decrypted ciphertext derived with 5000 iterations")
	}
}

func TestNewAEADWithIterations_BelowMinimum_ReturnsError(t *testing.T) {
	if _, err := NewAEADWithIterations(999); !errors.Is(err, ErrInvalidIterations) {
		t.Errorf("error = %v, want ErrInvalidIterations", err)
	}
}
