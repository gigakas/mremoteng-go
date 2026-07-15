package security

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestLegacy_RoundTrip_RecoversPlaintext(t *testing.T) {
	cases := []struct {
		name      string
		plaintext string
		password  string
	}{
		{"simple", "hello", "secret"},
		{"unicode", "contraseña: 日本語 ✓", "p@ssw0rd"},
		{"aligned block", "0123456789abcdef", "pw"},
		{"two blocks", "0123456789abcdef0123456789abcdef", "pw"},
		{"single byte", "x", "p"},
	}
	for _, c := range cases {
		enc, err := NewLegacy().Encrypt(c.plaintext, []byte(c.password))
		if err != nil {
			t.Errorf("%s: encrypt error: %v", c.name, err)
			continue
		}
		got, err := NewLegacy().Decrypt(enc, []byte(c.password))
		if err != nil {
			t.Errorf("%s: decrypt error: %v", c.name, err)
			continue
		}
		if got != c.plaintext {
			t.Errorf("%s: got %q, want %q", c.name, got, c.plaintext)
		}
	}
}

func TestLegacy_KeyIsMD5OfUTF8Password(t *testing.T) {
	// The legacy provider uses MD5(password); the same password therefore
	// yields a stable 128-bit key regardless of anything else.
	if want, got := len(legacyKey([]byte("pw"))), 16; want != got {
		t.Errorf("legacy key length = %d, want 16", got)
	}
}

func TestLegacy_EachEncryptionDiffers(t *testing.T) {
	a, _ := NewLegacy().Encrypt("same", []byte("pw"))
	b, _ := NewLegacy().Encrypt("same", []byte("pw"))
	if a == b {
		t.Error("two encryptions produced identical ciphertext; IV not random")
	}
}

func TestLegacy_EmptyInputs(t *testing.T) {
	if got, err := NewLegacy().Encrypt("", []byte("pw")); got != "" || err != nil {
		t.Errorf(`empty plaintext: got (%q,%v), want ("",nil)`, got, err)
	}
	if got, err := NewLegacy().Decrypt("", []byte("pw")); got != "" || err != nil {
		t.Errorf(`empty ciphertext: got (%q,%v), want ("",nil)`, got, err)
	}
	if _, err := NewLegacy().Encrypt("x", nil); !errors.Is(err, ErrEmptyPassword) {
		t.Errorf("nil password: got %v, want ErrEmptyPassword", err)
	}
}

func TestLegacy_MalformedInput_ReturnsError(t *testing.T) {
	if _, err := NewLegacy().Decrypt("!!!not base64!!!", []byte("pw")); err == nil {
		t.Error("expected error for invalid base64, got none")
	}
	short := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	if _, err := NewLegacy().Decrypt(short, []byte("pw")); err == nil {
		t.Error("expected error for ciphertext shorter than one IV block, got none")
	}
	// Valid base64 but the body is not a multiple of the block size.
	odd := base64.StdEncoding.EncodeToString(append([]byte("1234567890123456"), 'x'))
	if _, err := NewLegacy().Decrypt(odd, []byte("pw")); err == nil {
		t.Error("expected error for non-block-aligned body, got none")
	}
}

func TestProviders_CannotDecryptEachOther(t *testing.T) {
	enc, _ := NewAEAD().Encrypt("secret", []byte("pw"))
	if _, err := NewLegacy().Decrypt(enc, []byte("pw")); err == nil {
		t.Error("legacy provider decrypted AEAD ciphertext; cross-format must fail")
	}
	legacyEnc, _ := NewLegacy().Encrypt("secret", []byte("pw"))
	if _, err := NewAEAD().Decrypt(legacyEnc, []byte("pw")); err == nil {
		t.Error("aead provider decrypted legacy ciphertext; cross-format must fail")
	}
}

func TestProviders_ImplementInterface(t *testing.T) {
	var providers = []Provider{NewAEAD(), NewLegacy()}
	for _, p := range providers {
		ct, err := p.Encrypt("round trip", []byte("pw"))
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if got, err := p.Decrypt(ct, []byte("pw")); err != nil || got != "round trip" {
			t.Errorf("decrypt: got (%q,%v), want round trip", got, err)
		}
	}
}
