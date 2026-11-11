package credential_test

import (
	"testing"

	"github.com/mRemoteNG/mremoteng-go/internal/credential"
)

func TestFieldIs_CaseInsensitiveIdentifierMatch_ReturnsTrue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		first  string
		second string
		want   string
	}{
		{"display name match", "Username", "unrelated-slug", "username"},
		{"second identifier match", "f1", "PASSWORD", "password"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !credential.FieldIs(tc.first, tc.second, tc.want) {
				t.Errorf("FieldIs(%q, %q, %q) = false, want true", tc.first, tc.second, tc.want)
			}
		})
	}
}

func TestFieldIs_NeitherIdentifierMatches_ReturnsFalse(t *testing.T) {
	if credential.FieldIs("Notes", "notesPlain", "password") {
		t.Error(`FieldIs("Notes", "notesPlain", "password") = true, want false`)
	}
}
