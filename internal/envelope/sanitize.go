// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package envelope

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// DefaultMaxRunes caps a single untrusted field. Condition messages and
// provider errors are occasionally enormous, and an unbounded one floods the
// caller's context window (threat model, "Oversized logs/events").
const DefaultMaxRunes = 2048

// truncationMarker is appended to a truncated value so a reader can see that
// the text was cut rather than ending oddly on its own.
const truncationMarker = "…[truncated]"

// Sanitizer cleans untrusted cluster text before it becomes evidence.
//
// It does not try to decide whether text is malicious. That judgement is not
// available to us, and attempting it would produce false confidence. What it
// does is narrower and checkable: remove characters that let text lie about its
// own appearance, and bound its size.
//
// The separation that actually stops prompt injection is structural — untrusted
// text lives only in Evidence.Value and never reaches a server-authored field
// (ADR-0004). This is defence in depth on top of that.
type Sanitizer struct {
	// MaxRunes caps a single value. Zero means DefaultMaxRunes.
	MaxRunes int
}

// NewSanitizer returns a Sanitizer with the default limit.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{MaxRunes: DefaultMaxRunes}
}

// Result is the outcome of sanitizing one value, carrying the flags that end up
// on the Evidence item.
type Result struct {
	// Value is the cleaned text.
	Value string
	// Sanitized reports that at least one character was removed or replaced.
	Sanitized bool
	// Truncated reports that the value was cut to the limit.
	Truncated bool
	// OriginalLength is the rune count of the input as received, before any
	// normalisation or truncation.
	OriginalLength int
}

// Text sanitizes one untrusted string.
//
// It does four things, in order:
//
//  1. Replaces invalid UTF-8 with U+FFFD, so the result is always well-formed.
//  2. Normalises to NFC, so visually identical text has one representation and
//     comparisons in tests and rules are stable.
//  3. Removes Unicode format characters (category Cf) and control characters,
//     keeping only newline and tab. This covers bidi overrides (U+202A-202E,
//     U+2066-2069), zero-width characters (U+200B-200D), and the byte-order
//     mark — the tricks that let text render differently from how it reads.
//  4. Truncates to MaxRunes at a rune boundary.
func (s *Sanitizer) Text(in string) Result {
	res := Result{OriginalLength: utf8.RuneCountInString(in)}

	// utf8.RuneCountInString counts each invalid byte as one rune, which is the
	// honest count for a length the caller might compare against a raw payload.

	cleaned, replaced := replaceInvalidUTF8(in)
	normalized := norm.NFC.String(cleaned)

	removed := false

	// Normalise line endings before the rune scan. Doing it inside the loop
	// turns every CRLF into two newlines, because the '\n' following the '\r'
	// is then written as well.
	if strings.ContainsRune(normalized, '\r') {
		normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")
		removed = true
	}

	var b strings.Builder
	b.Grow(len(normalized))

	for _, r := range normalized {
		switch {
		case r == '\n' || r == '\t':
			// Deliberately kept: multi-line provider errors are much harder to
			// read as a single run-on line, and neither character can
			// misrepresent the text's meaning.
			b.WriteRune(r)

		case unicode.Is(unicode.Cf, r):
			// Format characters: bidi controls, zero-width joiners, the BOM,
			// soft hyphen. All of them let rendered text differ from its
			// character sequence.
			removed = true

		case unicode.IsControl(r):
			removed = true

		default:
			b.WriteRune(r)
		}
	}

	out := b.String()

	limit := s.MaxRunes
	if limit <= 0 {
		limit = DefaultMaxRunes
	}

	// The marker counts against the limit, so the result never exceeds MaxRunes.
	// A cap that its own marker can push past is not a cap.
	if utf8.RuneCountInString(out) > limit {
		if markerRunes := utf8.RuneCountInString(truncationMarker); limit > markerRunes {
			out = truncateRunes(out, limit-markerRunes) + truncationMarker
		} else {
			// Limit too small to fit the marker; drop it rather than overflow.
			out = truncateRunes(out, limit)
		}
		res.Truncated = true
	}

	res.Value = out
	res.Sanitized = removed || replaced
	return res
}

// Untrusted builds an Evidence item from raw cluster text, applying the
// sanitizer and setting the trust label and flags together.
//
// Using this rather than assembling an Evidence by hand is what keeps the
// invariant hard to break by accident: there is no path here that yields
// untrusted text without a Trust label on it.
func (s *Sanitizer) Untrusted(ev Evidence, raw string) Evidence {
	res := s.Text(raw)

	ev.Trust = TrustUntrusted
	ev.Value = res.Value
	ev.Sanitized = res.Sanitized
	ev.Truncated = res.Truncated

	// Only record the original length when it tells the reader something the
	// value does not already show.
	if res.Truncated {
		ev.OriginalLength = res.OriginalLength
	}

	return ev
}

// replaceInvalidUTF8 substitutes U+FFFD for any byte sequence that is not valid
// UTF-8, reporting whether it changed anything.
//
// Go strings are byte slices and may hold arbitrary bytes. Kubernetes fields
// are meant to be UTF-8, but "meant to be" is not a guarantee worth relying on
// for attacker-influenced data.
func replaceInvalidUTF8(s string) (string, bool) {
	if utf8.ValidString(s) {
		return s, false
	}

	var b strings.Builder
	b.Grow(len(s))

	for i, r := range s {
		if r == utf8.RuneError {
			// range yields RuneError for an invalid byte; distinguish a real
			// U+FFFD in the input from a decoding failure by re-decoding.
			if _, width := utf8.DecodeRuneInString(s[i:]); width == 1 {
				b.WriteRune(utf8.RuneError)
				continue
			}
		}
		b.WriteRune(r)
	}

	return b.String(), true
}

// truncateRunes cuts s to at most n runes, never splitting one.
func truncateRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
