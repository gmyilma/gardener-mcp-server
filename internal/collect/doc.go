// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package collect holds the read-only, bounded collectors, one subpackage per
// cluster role.
//
// Every collector reports its own outcome (ok, forbidden, notFound, timeout,
// skipped, error) so that a partial diagnosis still returns findings and can
// say which layer is missing (architecture-review.md §9).
package collect
