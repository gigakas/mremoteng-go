package security

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

// RFC 6070 PBKDF2-HMAC-SHA1 reference vectors for ("password","salt"),
// dkLen=20. They confirm pbkdf2.Key with sha1.New is the standard PRF, i.e.
// the same one BouncyCastle's Pkcs5S2ParametersGenerator (default SHA-1
// digest) applies in mRemoteNG.
var rfc6070 = []struct {
	iter    int
	wantHex string
}{
	{1, "0c60c80f961f0e71f3a9b524af6012062fe037a6"},
	{2, "ea6c014dc72d6f8ccd1ed92ace1d41f0d8de8957"},
	{4096, "4b007901b765489abead49d926f721d065a429c1"},
}

func TestPBKDF2_MatchesRFC6070(t *testing.T) {
	for _, c := range rfc6070 {
		got := hex.EncodeToString(pbkdf2.Key(
			[]byte("password"), []byte("salt"), c.iter, 20, sha1.New))
		if got != c.wantHex {
			t.Errorf("iter=%d: got %s, want %s", c.iter, got, c.wantHex)
		}
	}
}

// deriveAEADKey wraps PBKDF2-HMAC-SHA1 with mRemoteNG's parameters (1000
// iterations, 256-bit key) over the Latin-1 password encoding. For an ASCII
// password the encoding is a pass-through, so the result must equal a direct
// PBKDF2 call with the same parameters.
func TestDeriveAEADKey_UsesPBKDF2SHA1_1000Iterations(t *testing.T) {
	key := deriveAEADKey([]byte("password"), []byte("salt"))
	if len(key) != aeadKeyBytes {
		t.Fatalf("key length = %d, want %d", len(key), aeadKeyBytes)
	}
	direct := pbkdf2.Key([]byte("password"), []byte("salt"),
		pbkdf2Iterations, aeadKeyBytes, sha1.New)
	if !bytes.Equal(key, direct) {
		t.Errorf("deriveAEADKey diverged from direct PBKDF2-SHA1/1000:\n got %x\nwant %x", key, direct)
	}
}

func TestPkcs5PasswordToBytes_Ascii_MatchesRawBytes(t *testing.T) {
	got := pkcs5PasswordToBytes("password")
	want := []byte("password")
	if !bytes.Equal(got, want) {
		t.Errorf("got %x, want %x", got, want)
	}
}

func TestPkcs5PasswordToBytes_Latin1_TakesLowByte(t *testing.T) {
	// 'é' is U+00E9: one UTF-16 unit, low byte 0xE9 (the ISO-8859-1 byte).
	got := pkcs5PasswordToBytes("café")
	want := []byte{'c', 'a', 'f', 0xE9}
	if !bytes.Equal(got, want) {
		t.Errorf("got %x, want %x", got, want)
	}
}

func TestPkcs5PasswordToBytes_Astral_LowByteOfSurrogate(t *testing.T) {
	// '€' is U+20AC: one UTF-16 unit 0x20AC, low byte 0xAC.
	got := pkcs5PasswordToBytes("a€b")
	want := []byte{'a', 0xAC, 'b'}
	if !bytes.Equal(got, want) {
		t.Errorf("got %x, want %x", got, want)
	}
}

func TestPkcs7_RoundTrip(t *testing.T) {
	cases := [][]byte{
		{},
		{'a'},
		bytes.Repeat([]byte{'a'}, 15),
		bytes.Repeat([]byte{'a'}, 16),
		bytes.Repeat([]byte{'a'}, 17),
		bytes.Repeat([]byte{'x'}, 256),
	}
	for _, in := range cases {
		padded := pkcs7Pad(in, 16)
		if len(padded)%16 != 0 {
			t.Errorf("padded length %d is not block-aligned", len(padded))
		}
		if len(in) > 0 && len(in)%16 == 0 && len(padded) == len(in) {
			t.Errorf("aligned input did not get a full extra padding block")
		}
		out, err := pkcs7Unpad(padded, 16)
		if err != nil {
			t.Errorf("unpad error: %v", err)
			continue
		}
		if !bytes.Equal(out, in) {
			t.Errorf("round trip mismatch: got %q, want %q", out, in)
		}
	}
}

func TestPkcs7Unpad_RejectsInvalid(t *testing.T) {
	cases := map[string][]byte{
		"empty":             {},
		"not block-aligned": bytes.Repeat([]byte{1}, 17),
		"zero pad length":   bytes.Repeat([]byte{0}, 16),
		"pad length > blk":  append(bytes.Repeat([]byte{'a'}, 16), 17),
		"inconsistent pad":  append([]byte("aaaaaaaaaaaaaaa"), 2),
	}
	for name, in := range cases {
		if _, err := pkcs7Unpad(in, 16); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
	}
}
