package credential

import "strings"

// FieldIs reports whether either of the two identifiers a response field
// carries matches want, ignoring case. Providers had this predicate twice
// under the name fieldIs: the field's display name varies (label, slug,
// fieldName) but also its id, and the secret-store templates do not agree
// on casing, so both are matched case-insensitively.
func FieldIs(first, second, want string) bool {
	return strings.EqualFold(first, want) || strings.EqualFold(second, want)
}
