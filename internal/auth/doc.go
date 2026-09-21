// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package auth authenticates the *caller* in service mode, via mTLS or a
// bearer token.
//
// It never forwards a caller token onward to Kubernetes. The MCP authorization
// spec forbids token pass-through, and doing so would defeat the single-tenant
// scoping that service mode relies on (architecture-review.md §3).
package auth
