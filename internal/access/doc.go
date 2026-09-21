// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package access is the credential broker. It hands out ready-to-use clients
// for the Garden, Seed and Shoot, and caches short-lived viewer kubeconfigs.
//
// It returns clients, never credential bytes. No kubeconfig, token or client
// certificate may leave this package — that invariant is what keeps credentials
// out of MCP responses, logs and test snapshots (architecture-review.md §3).
//
// Requesting shoots/adminkubeconfig is not implemented and must not be.
package access
