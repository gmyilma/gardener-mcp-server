// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

// Command gardener-mcp-server exposes Gardener-aware diagnostics over the
// Model Context Protocol.
//
// The binary runs in one of two deployment modes (architecture-review.md §3):
//
//	local   stdio transport, using the caller's own Garden kubeconfig.
//	        Their RBAC decides what is visible, so there is no confused deputy.
//	service streamable HTTP, using the server's own ServiceAccount. Single
//	        tenant: one persona and one project scope per deployment.
//
// Service mode arrives in M1.6.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/gmyilma/gardener-mcp-server/internal/audit"
	"github.com/gmyilma/gardener-mcp-server/internal/policy"
	"github.com/gmyilma/gardener-mcp-server/internal/server"
	"github.com/gmyilma/gardener-mcp-server/internal/tools/serverinfo"
)

// Injected at build time via -ldflags; see the Makefile.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// config holds the command-line configuration. Keeping it as a plain struct
// populated by a function, rather than reading globals, is what makes the
// wiring testable.
//
// Go note: this is the idiomatic alternative to a DI container. Dependencies
// are passed explicitly down from main rather than resolved from a registry —
// roughly what you would do with constructor injection in Java, minus Spring.
type config struct {
	mode        string
	persona     string
	projects    string
	showVersion bool
}

func parseFlags(args []string, stderr *os.File) (*config, error) {
	fs := flag.NewFlagSet("gardener-mcp-server", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &config{}
	fs.StringVar(&cfg.mode, "mode", "local", "deployment mode: local (stdio) or service (http)")
	fs.StringVar(&cfg.persona, "persona", "shoot-owner", "persona: shoot-owner or operator")
	fs.StringVar(&cfg.projects, "projects", "", "comma-separated Gardener project allowlist (service mode)")
	fs.BoolVar(&cfg.showVersion, "version", false, "print version information and exit")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate rejects unknown enum values before anything else runs. The policy
// layer enforces the same values again at request time; this is only the
// startup check.
func (c *config) validate() error {
	switch c.mode {
	case "local", "service":
	default:
		return fmt.Errorf("invalid --mode %q: want local or service", c.mode)
	}

	switch c.persona {
	case "shoot-owner", "operator":
	default:
		return fmt.Errorf("invalid --persona %q: want shoot-owner or operator", c.persona)
	}

	return nil
}

// projectList splits the allowlist flag. An empty flag yields nil, meaning
// "no restriction beyond the credential's own RBAC".
func (c *config) projectList() []string {
	if strings.TrimSpace(c.projects) == "" {
		return nil
	}
	parts := strings.Split(c.projects, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gardener-mcp-server: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches: parse flags, handle --version, otherwise serve.
//
// Returning an error rather than calling os.Exit is what makes this reachable
// from tests.
func run(args []string) error {
	cfg, err := parseFlags(args, os.Stderr)
	if err != nil {
		return err
	}

	if cfg.showVersion {
		fmt.Printf("gardener-mcp-server %s\n  commit:     %s\n  built:      %s\n  go:         %s\n",
			version, commit, buildDate, runtime.Version())
		return nil
	}

	return serve(cfg)
}

// serve validates the configuration, wires the layers together and runs the
// MCP server until the client disconnects.
func serve(cfg *config) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	if cfg.mode == "service" {
		return errors.New("service mode is not implemented yet: it arrives in M1.6")
	}

	pol, err := policy.New(policy.Config{
		Persona:  policy.Persona(cfg.persona),
		Projects: cfg.projectList(),
	})
	if err != nil {
		return fmt.Errorf("policy: %w", err)
	}

	// The audit log goes to stderr, never stdout.
	//
	// In stdio mode stdout *is* the MCP transport: it carries newline-delimited
	// JSON-RPC. A single stray line written there corrupts the protocol stream,
	// and the client then fails in a way that looks nothing like a logging bug.
	srv, err := server.New(server.Options{
		Version: version,
		Policy:  pol,
		Audit:   audit.New(os.Stderr),
	})
	if err != nil {
		return err
	}

	serverinfo.Register(srv, serverinfo.Config{
		Version:  version,
		Persona:  cfg.persona,
		Mode:     cfg.mode,
		Projects: cfg.projectList(),
		Tools:    []string{"server_info"},
	})

	// Stop cleanly on Ctrl-C or SIGTERM so the client sees a closed transport
	// rather than a truncated frame.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.ServeStdio(ctx)
}
