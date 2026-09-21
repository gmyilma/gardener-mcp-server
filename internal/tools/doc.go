// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package tools holds one subpackage per MCP tool, each with its input and
// output types and its handler.
//
// Rules that apply to every tool (architecture-review.md §8):
//
//   - Inputs are project and shoot plus enums and bounded integers. Never
//     namespaces, object names or cluster names — those are derived on the
//     server, which is what stops one tool's output becoming another's
//     unvalidated input.
//   - Outputs are curated fields in the shared envelope, never raw objects.
//   - Handlers re-validate their input. The client having honoured the schema
//     is never assumed.
package tools
