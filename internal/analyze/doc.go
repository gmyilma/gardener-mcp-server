// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package analyze turns collected facts into findings using deterministic
// rules with stable IDs.
//
// There is no LLM in this path. Severity comes from the rule, and confidence is
// an enum the rule declares rather than a computed number, so that output stays
// reproducible and testable (architecture-review.md §8, §9).
package analyze
