package util

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func ToSlug(s string) string {
	// Normalize to decomposed form (NFD)
	normalized := norm.NFD.String(s)

	var b strings.Builder
	b.Grow(len(normalized))

	prevDash := false
	prevLowerOrDigit := false

	for _, r := range normalized {

		// Strip diacritic marks
		if unicode.Is(unicode.Mn, r) {
			continue
		}

		switch {
		case unicode.IsUpper(r):
			if prevLowerOrDigit && !prevDash {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
			prevLowerOrDigit = true

		case unicode.IsLower(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
			prevLowerOrDigit = true

		case r == '_' || r == ' ' || r == '-':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
			prevLowerOrDigit = false

		default:
			// ignore everything else (symbols, punctuation, non-ascii letters)
		}
	}

	result := strings.Trim(b.String(), "-")

	return result
}

// NewShortID generates a 7-character URL-safe random string.
func NewShortID() string {
	// 7 base64 chars = 42 bits
	// Generate 6 bytes (48 bits) and trim to 7 chars.
	b := make([]byte, 6)

	_, _ = io.ReadFull(rand.Reader, b)

	s := base64.RawURLEncoding.EncodeToString(b)

	return s[:7]
}
