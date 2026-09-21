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
// M0 wires up flags and version reporting only. The MCP server itself arrives
// in M1.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

// Injected at build time via -ldflags; see the Makefile.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// config holds the command-line configuration. Keeping it as a plain struct
// populated by a function, rather than reading globals, is what makes the
// wiring testable later.
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
// layer in M1 will enforce the same values again at request time; this is only
// the startup check.
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

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gardener-mcp-server: %v\n", err)
		os.Exit(1)
	}
}

// run holds the real logic so that main stays a thin shell around it. Returning
// an error instead of calling os.Exit is what makes this reachable from tests.
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

	if err := cfg.validate(); err != nil {
		return err
	}

	return fmt.Errorf("mode %q is not implemented yet: the MCP server arrives in M1", cfg.mode)
}
