// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strings"
	"testing"
)

// Go note: this is a "table-driven test", the standard Go idiom for what you
// would write with @ParameterizedTest in JUnit or @pytest.mark.parametrize.
// The cases are plain data, and t.Run gives each one its own named subtest.
func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config
		wantErr string // substring; empty means no error expected
	}{
		{
			name: "defaults are valid",
			cfg:  config{mode: "local", persona: "shoot-owner"},
		},
		{
			name: "operator in service mode is valid",
			cfg:  config{mode: "service", persona: "operator"},
		},
		{
			name:    "unknown mode is rejected",
			cfg:     config{mode: "cluster", persona: "operator"},
			wantErr: `invalid --mode "cluster"`,
		},
		{
			// A typo must not silently fall back to the narrower persona, and
			// must never be treated as "operator". Persona decides whether Seed
			// tools are reachable at all (architecture-review.md §3).
			name:    "unknown persona is rejected",
			cfg:     config{mode: "local", persona: "admin"},
			wantErr: `invalid --persona "admin"`,
		},
		{
			name:    "empty persona is rejected",
			cfg:     config{mode: "local", persona: ""},
			wantErr: `invalid --persona ""`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.validate()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validate() = nil, want error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("validate() = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseFlagsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := parseFlags(nil, os.Stderr)
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	// The safe default matters: local mode uses the caller's own credentials,
	// and shoot-owner is the persona without Seed access.
	if cfg.mode != "local" {
		t.Errorf("default mode = %q, want %q", cfg.mode, "local")
	}
	if cfg.persona != "shoot-owner" {
		t.Errorf("default persona = %q, want %q", cfg.persona, "shoot-owner")
	}
	if cfg.projects != "" {
		t.Errorf("default projects = %q, want empty", cfg.projects)
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	if err := run([]string{"--version"}); err != nil {
		t.Fatalf("run(--version) = %v, want nil", err)
	}
}

func TestRunRejectsInvalidMode(t *testing.T) {
	t.Parallel()

	err := run([]string{"--mode=bogus"})
	if err == nil {
		t.Fatal("run(--mode=bogus) = nil, want error")
	}
}

func TestProjectList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "whitespace only", in: "  ", want: nil},
		{name: "single", in: "alpha", want: []string{"alpha"}},
		{name: "several", in: "alpha,beta", want: []string{"alpha", "beta"}},
		{name: "spaces are trimmed", in: " alpha , beta ", want: []string{"alpha", "beta"}},
		{name: "empty entries are dropped", in: "alpha,,beta,", want: []string{"alpha", "beta"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := (&config{projects: tc.in}).projectList()

			if len(got) != len(tc.want) {
				t.Fatalf("projectList(%q) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("projectList(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestServeRejectsServiceMode(t *testing.T) {
	t.Parallel()

	// Service mode is not implemented; it must fail loudly rather than
	// silently falling back to stdio.
	err := serve(&config{mode: "service", persona: "operator"})
	if err == nil {
		t.Fatal("serve(service) = nil, want an error")
	}
	if !strings.Contains(err.Error(), "M1.6") {
		t.Errorf("error = %q, want it to say when service mode arrives", err)
	}
}

func TestServeRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	if err := serve(&config{mode: "local", persona: "admin"}); err == nil {
		t.Error("serve() with an unknown persona = nil, want an error")
	}
	if err := serve(&config{mode: "bogus", persona: "operator"}); err == nil {
		t.Error("serve() with an unknown mode = nil, want an error")
	}
}

func TestServeRejectsInvalidProjectName(t *testing.T) {
	t.Parallel()

	// A malformed allowlist entry must fail at startup, not per request.
	err := serve(&config{mode: "local", persona: "operator", projects: "NOT OK"})
	if err == nil {
		t.Fatal("serve() with an invalid project name = nil, want an error")
	}
	if !strings.Contains(err.Error(), "policy") {
		t.Errorf("error = %q, want it to name the policy layer", err)
	}
}
