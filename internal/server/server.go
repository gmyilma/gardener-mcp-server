// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gmyilma/gardener-mcp-server/internal/audit"
	"github.com/gmyilma/gardener-mcp-server/internal/policy"
)

// Options configures a Server. Every field except Version is required.
type Options struct {
	// Version is reported to clients during initialization.
	Version string

	// Policy gates every tool call before it reaches a handler.
	Policy *policy.Policy

	// Audit receives one record per tool call, including refused ones.
	Audit *audit.Logger
}

// Server is the MCP protocol surface.
//
// The go-sdk is reachable only from this package. Tool packages depend on
// Register and Handler below, which mention no SDK type, so replacing the SDK
// means rewriting this file and nothing else (ADR-0002).
type Server struct {
	mcp    *mcp.Server
	policy *policy.Policy
	audit  *audit.Logger
}

// New builds a Server with no tools registered.
func New(opts Options) (*Server, error) {
	if opts.Policy == nil {
		return nil, errors.New("server: Policy is required")
	}
	if opts.Audit == nil {
		return nil, errors.New("server: Audit is required")
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}

	impl := &mcp.Implementation{
		Name:    "gardener-mcp-server",
		Title:   "Gardener diagnostics",
		Version: opts.Version,
		Description: "Gardener-aware diagnostics over MCP. Resolves Garden to Seed to Shoot " +
			"topology and returns curated findings with evidence. Makes no changes to any cluster.",
	}

	return &Server{
		mcp:    mcp.NewServer(impl, nil),
		policy: opts.Policy,
		audit:  opts.Audit,
	}, nil
}

// ServeStdio runs the server over stdin/stdout until the client disconnects or
// ctx is cancelled. This is local mode (ADR-0001): the caller's own credentials,
// their own RBAC, no shared identity.
//
// Note for callers: in this mode stdout carries the JSON-RPC stream. Anything
// else written there corrupts the protocol, so logs must go to stderr.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.ServeTransport(ctx, &mcp.StdioTransport{})
}

// ServeTransport runs the server over an arbitrary transport.
//
// This is the one place an SDK type appears in an exported signature, and
// deliberately so: a transport is inherently protocol-shaped, and pretending
// otherwise would mean inventing an abstraction with exactly one useful
// implementation. What ADR-0002 asks to keep SDK-free is the *tool* surface —
// ToolDef and Handler — and that holds.
//
// Used by tests, and by service mode's streamable HTTP transport in M1.6.
func (s *Server) ServeTransport(ctx context.Context, t mcp.Transport) error {
	return s.mcp.Run(ctx, t)
}

// ToolDef describes a tool independently of the SDK.
//
// v0.1 tools are all read-only, idempotent and closed-world, so those hints are
// not configurable here. They are metadata for clients, never security
// controls — the policy layer is the control.
type ToolDef struct {
	// Name is the MCP tool name, e.g. "diagnose_shoot".
	Name string
	// Title is a human-readable display name.
	Title string
	// Description tells the model what the tool is for.
	Description string
}

// Handler is a tool implementation in this project's terms.
//
// Go note: this is a generic function type. In Java you might express it as a
// BiFunction with type parameters; the difference is that Go infers In and Out
// at the call site, and the SDK derives the JSON schema from them, so the
// declared Go types and the published contract cannot drift apart.
type Handler[In, Out any] func(ctx context.Context, in In) (Out, error)

// ShootTarget is implemented by tool inputs that name a Shoot. Inputs that
// implement it get their target recorded in the audit log and authorized
// before the handler runs.
//
// Defining the interface here, next to the code that consumes it, rather than
// next to the inputs that satisfy it, is the Go convention.
type ShootTarget interface {
	// Target returns the project and shoot named by this input.
	Target() (project, shoot string)
}

// Register adds a tool to the server, wrapping it with authorization and
// auditing so that no handler can be reached without both.
func Register[In, Out any](s *Server, def ToolDef, h Handler[In, Out]) {
	tool := &mcp.Tool{
		Name:        def.Name,
		Description: def.Description,
		Annotations: &mcp.ToolAnnotations{
			Title:          def.Title,
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  ptr(false),
		},
	}

	mcp.AddTool(s.mcp, tool, func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var zero Out
		start := time.Now()

		ev := audit.Event{
			Tool:    def.Name,
			Persona: string(s.policy.Persona()),
		}

		// Authorize before anything else. An input that names a Shoot is
		// validated and checked against the project allowlist here, so a
		// handler never sees an unauthorized target.
		if t, ok := any(in).(ShootTarget); ok {
			project, shoot := t.Target()
			ev.Project, ev.Shoot = project, shoot

			if err := s.policy.AuthorizeShoot(project, shoot); err != nil {
				ev.Outcome = outcomeFor(err)
				ev.Reason = reasonFor(err)
				ev.Duration = time.Since(start)
				s.audit.Record(ctx, ev)

				// The error text is server-authored: it names the policy
				// decision, never an upstream message.
				return nil, zero, fmt.Errorf("%s: %w", def.Name, err)
			}
		}

		// Bound the call. Without this a handler could run until the client
		// gives up, holding cluster connections open.
		ctx, cancel := context.WithTimeout(ctx, s.policy.Budget().Deadline)
		defer cancel()

		out, err := h(ctx, in)

		ev.Duration = time.Since(start)
		if err != nil {
			ev.Outcome = outcomeFor(err)
			ev.Reason = reasonFor(err)
			s.audit.Record(ctx, ev)
			return nil, zero, err
		}

		ev.Outcome = audit.OutcomeOK
		s.audit.Record(ctx, ev)

		// Returning a nil CallToolResult lets the SDK build one from Out,
		// including the structuredContent and its schema.
		return nil, out, nil
	})
}

// outcomeFor maps an error to an audit outcome.
func outcomeFor(err error) audit.Outcome {
	switch {
	case errors.Is(err, policy.ErrInvalidIdentifier):
		return audit.OutcomeInvalid
	case errors.Is(err, policy.ErrProjectNotAllowed),
		errors.Is(err, policy.ErrPersonaDenied),
		errors.Is(err, policy.ErrNamespaceNotPinned):
		return audit.OutcomeDenied
	case errors.Is(err, context.DeadlineExceeded):
		return audit.OutcomeTimeout
	default:
		return audit.OutcomeError
	}
}

// reasonFor produces a short, server-authored reason for the audit log.
//
// It deliberately does not use err.Error(). An upstream error can embed text
// from a cluster, and cluster text does not belong in an audit trail any more
// than it belongs in a finding (ADR-0004).
func reasonFor(err error) string {
	switch {
	case errors.Is(err, policy.ErrInvalidIdentifier):
		return "invalid identifier"
	case errors.Is(err, policy.ErrProjectNotAllowed):
		return "project not allowed"
	case errors.Is(err, policy.ErrPersonaDenied):
		return "persona denied"
	case errors.Is(err, policy.ErrNamespaceNotPinned):
		return "seed namespace not pinned"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline exceeded"
	default:
		return "internal error"
	}
}

func ptr[T any](v T) *T { return &v }
