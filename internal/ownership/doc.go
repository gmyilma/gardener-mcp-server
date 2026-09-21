// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package ownership detects whether a Shoot is managed by ArgoCD, Flux or
// directly, from annotations, labels and managedFields.
//
// Absence of markers means Unknown, never Direct. A repository is only ever
// taken from explicit configuration and is never inferred, because Argo and
// Flux usually run outside the virtual Garden (architecture-review.md §11).
package ownership
