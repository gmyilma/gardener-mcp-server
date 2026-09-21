// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package seed collects control-plane state from a Seed: extension resources
// and their lastError, ManagedResource conditions, Etcd health, and MCM
// machines.
//
// Operator persona only, and always pinned to one technical namespace. Reads
// are limited to status fields; pod specs and logs are deliberately excluded
// because they leak infrastructure detail (architecture-review.md §4).
package seed
