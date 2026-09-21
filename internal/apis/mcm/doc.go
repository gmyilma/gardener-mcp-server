// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package mcm holds minimal local types for machine-controller-manager
// resources.
//
// MCM publishes no separate API module, and its root module pins an apiserver
// and code-generator tree that would dominate this project's dependency graph.
// These types are decoded from unstructured objects and kept honest by contract
// tests against MCM's CRDs (architecture-review.md §7, option E).
package mcm
