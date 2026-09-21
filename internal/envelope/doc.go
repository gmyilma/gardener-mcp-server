// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package envelope defines the shared response shape (meta, target, findings,
// evidence, sources) and the sanitizer that guards it.
//
// The central invariant: untrusted cluster text appears only as
// evidence[].value, and is never interpolated into server-authored fields such
// as title or interpretation. Server-written text contains templates and
// validated identifiers only.
//
// The sanitizer normalises to NFC, strips control and bidi characters, and
// truncates over-long values, recording originalLength (architecture-review.md
// §4, §9).
package envelope
