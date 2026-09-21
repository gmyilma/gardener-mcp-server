// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package shoot collects Node conditions and kube-system pod readiness from a
// Shoot cluster, using a short-lived viewer kubeconfig.
//
// Everything here is tenant-controlled and therefore untrusted.
package shoot
