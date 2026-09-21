// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package topology resolves a ShootRef into the concrete clusters a diagnosis
// must visit: the Garden, the Seed and its control-plane namespace, and the
// Shoot API server.
//
// The namespace always comes from Shoot.status.technicalID and is never
// recomputed from project and Shoot names: ComputeTechnicalID still honours
// older patterns for backwards compatibility, so a reconstructed name can be
// wrong (architecture-review.md §5, row 2).
//
// When spec.seedName and status.seedName differ, a control-plane migration is
// in progress and both are reported.
package topology
