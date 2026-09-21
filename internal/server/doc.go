// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package server owns the MCP protocol surface: go-sdk setup, the stdio and
// streamable-HTTP transports, and the tool registry.
//
// The go-sdk is kept behind a small internal registry interface so that
// swapping SDKs stays cheap (architecture-review.md §6).
package server
