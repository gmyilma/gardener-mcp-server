// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package policy decides whether a tool call may proceed, before any cluster
// is contacted.
//
// It enforces persona gating, the project allowlist, per-caller rate limits and
// budgets (deadlines and object caps). It also pins every Seed read to the
// resolved Shoot status.technicalID, which is what prevents cross-Shoot
// disclosure through a Seed (architecture-review.md §3, §4).
package policy
