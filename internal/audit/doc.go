// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package audit records structured entries for every tool call: caller, tool,
// arguments, clusters contacted and outcome.
//
// Credentials must never reach this package. Log redaction is tested, not
// assumed (architecture-review.md §4).
package audit
