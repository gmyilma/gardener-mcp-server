// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Package serverinfo implements the server_info tool.
//
// It reports what this deployment is and what it is permitted to do. That is
// worth exposing rather than leaving implicit: a caller can confirm the persona
// and the absence of a mutation path before trusting anything else the server
// says, and an operator can tell two deployments apart.
//
// It contacts no cluster, so it also serves as the end-to-end check that the
// MCP plumbing works.
package serverinfo

import (
	"context"

	"github.com/gmyilma/gardener-mcp-server/internal/server"
)

// Input takes no arguments.
//
// Go note: an empty struct is how Go expresses "no parameters" for a generic
// handler. The SDK infers an empty JSON schema from it, so a client sending
// unexpected arguments is rejected rather than silently ignored.
type Input struct{}

// Output describes the deployment.
type Output struct {
	Name    string `json:"name"`
	Version string `json:"version"`

	// Persona is fixed for the life of the deployment. A shoot-owner
	// deployment attempts no Seed call at all.
	Persona string `json:"persona"`

	// Mode is "local" (stdio, the caller's own credentials) or "service"
	// (HTTP, the server's ServiceAccount, single-tenant).
	Mode string `json:"mode"`

	// Projects is the configured allowlist. Empty means the deployment relies
	// on the credential's own RBAC.
	Projects []string `json:"projects,omitempty"`

	// NoResourceMutationPath is always true and is stated explicitly so a
	// caller need not infer it. The server holds no write credential.
	//
	// Note the wording: requesting a viewer kubeconfig uses the Kubernetes
	// create verb on a subresource, so "read-only" would be literally false
	// (ADR-0005).
	NoResourceMutationPath bool `json:"noResourceMutationPath"`

	// Tools lists the tool names this deployment exposes.
	Tools []string `json:"tools"`
}

// Config is what the tool reports. It is captured at registration, so the
// answer cannot drift from how the process was actually started.
type Config struct {
	Version  string
	Persona  string
	Mode     string
	Projects []string
	Tools    []string
}

// Register adds server_info to s.
func Register(s *server.Server, cfg Config) {
	out := Output{
		Name:                   "gardener-mcp-server",
		Version:                cfg.Version,
		Persona:                cfg.Persona,
		Mode:                   cfg.Mode,
		Projects:               cfg.Projects,
		NoResourceMutationPath: true,
		Tools:                  cfg.Tools,
	}

	server.Register(s, server.ToolDef{
		Name:  "server_info",
		Title: "Server information",
		Description: "Report this deployment's version, persona, mode and the tools it exposes. " +
			"Contacts no cluster. Use it to confirm what the server is permitted to do.",
	}, func(_ context.Context, _ Input) (Output, error) {
		return out, nil
	})
}
