// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Persona decides how much of Gardener a deployment may see. It is fixed per
// deployment by a startup flag, never taken from a request, so a tool argument
// cannot widen it (ADR-0001).
type Persona string

// The two personas supported in v0.1. Multi-tenant per-caller identity is
// deliberately deferred.
const (
	// PersonaShootOwner sees only the Garden layer. No Seed tool is reachable,
	// and no Seed call is attempted — not even one that would fail.
	PersonaShootOwner Persona = "shoot-owner"

	// PersonaOperator additionally sees Seed control-plane state, pinned to one
	// Shoot's technical namespace at a time.
	PersonaOperator Persona = "operator"
)

// Budget bounds the work a single tool call may do. Without these, one caller
// looping over diagnose_shoot can put real load on a Seed's API server.
type Budget struct {
	// Deadline is the wall-clock limit for a whole tool call.
	Deadline time.Duration
	// PerCollector is the limit for one collector against one cluster.
	PerCollector time.Duration
	// MaxConcurrentPerCluster caps parallel requests to any single cluster.
	MaxConcurrentPerCluster int
	// MaxEvents, MaxMachines and MaxPods cap list sizes. Exceeding a cap sets
	// truncated on the result rather than failing the call.
	MaxEvents   int
	MaxMachines int
	MaxPods     int
}

// DefaultBudget matches the values in architecture-review.md §9.
func DefaultBudget() Budget {
	return Budget{
		Deadline:                20 * time.Second,
		PerCollector:            5 * time.Second,
		MaxConcurrentPerCluster: 4,
		MaxEvents:               50,
		MaxMachines:             500,
		MaxPods:                 200,
	}
}

// maxDeadline is the ceiling a caller-supplied deadline is clamped to.
const maxDeadline = 45 * time.Second

// Config is the deployment's fixed policy. It is validated once at startup.
type Config struct {
	Persona Persona

	// Projects is an allowlist of Gardener project names. Empty means "no
	// project restriction beyond what the credential's own RBAC allows", which
	// is the sensible default for local mode where the caller uses their own
	// kubeconfig.
	//
	// In service mode this should always be set: the deployment's whole view is
	// exposed to anyone who can reach the endpoint (ADR-0001).
	Projects []string

	Budget Budget
}

// Policy answers "may this call proceed, and within what bounds". It holds no
// cluster clients and performs no I/O, which is what makes it exhaustively
// testable.
//
// Go note: this is a plain struct with a constructor returning an error, rather
// than an interface. Go convention is to define interfaces where they are
// consumed, not where they are implemented — so any interface over this belongs
// in the packages that call it, not here.
type Policy struct {
	persona  Persona
	projects []string
	budget   Budget
}

// Errors returned by Policy. Callers compare with errors.Is rather than on
// message text.
var (
	// ErrPersonaDenied means the deployment's persona has no access to this
	// capability at all.
	ErrPersonaDenied = errors.New("persona denied")
	// ErrProjectNotAllowed means the project is outside this deployment's scope.
	ErrProjectNotAllowed = errors.New("project not allowed")
	// ErrInvalidIdentifier means an argument failed validation.
	ErrInvalidIdentifier = errors.New("invalid identifier")
	// ErrNamespaceNotPinned means a Seed read was attempted outside the
	// resolved Shoot's technical namespace.
	ErrNamespaceNotPinned = errors.New("seed namespace not pinned to resolved shoot")
)

// New validates a Config and returns the Policy it describes.
func New(cfg Config) (*Policy, error) {
	switch cfg.Persona {
	case PersonaShootOwner, PersonaOperator:
	default:
		return nil, fmt.Errorf("%w: persona %q", ErrInvalidIdentifier, cfg.Persona)
	}

	for _, p := range cfg.Projects {
		if err := ValidateIdentifier("project", p); err != nil {
			return nil, err
		}
	}

	b := cfg.Budget
	if b.Deadline <= 0 {
		b = DefaultBudget()
	}
	if b.Deadline > maxDeadline {
		b.Deadline = maxDeadline
	}

	// Copy the slice so a later mutation by the caller cannot change policy
	// after startup.
	return &Policy{
		persona:  cfg.Persona,
		projects: slices.Clone(cfg.Projects),
		budget:   b,
	}, nil
}

// Persona reports the deployment's fixed persona.
func (p *Policy) Persona() Persona { return p.persona }

// Budget reports the effective budget, already clamped.
func (p *Policy) Budget() Budget { return p.budget }

// SeedAccessAllowed reports whether this deployment may read Seed state at all.
//
// A shoot-owner deployment must not merely fail a Seed call — it must not make
// one. Seed data covers every tenant on that Seed, so the correct behaviour is
// to record the layer as skipped and carry on (architecture-review.md §9).
func (p *Policy) SeedAccessAllowed() bool {
	return p.persona == PersonaOperator
}

// AuthorizeShoot checks a tool call's target before any cluster is contacted.
//
// It validates the identifiers, because tool arguments are the untrusted
// boundary (threat model TB1) and the go-sdk having validated against a schema
// is a convenience, not a security control.
func (p *Policy) AuthorizeShoot(project, shoot string) error {
	if err := ValidateIdentifier("project", project); err != nil {
		return err
	}
	if err := ValidateIdentifier("shoot", shoot); err != nil {
		return err
	}
	return p.authorizeProject(project)
}

// authorizeProject enforces the allowlist. An empty allowlist permits any
// project, leaving the credential's own RBAC as the only limit.
func (p *Policy) authorizeProject(project string) error {
	if len(p.projects) == 0 {
		return nil
	}
	if slices.Contains(p.projects, project) {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrProjectNotAllowed, project)
}

// AuthorizeSeedRead is the namespace pin.
//
// The seed-reader ClusterRole can read every control-plane namespace on a Seed,
// so RBAC alone does not stop cross-Shoot disclosure. This is the control that
// does: a Seed read is permitted only in the namespace named by the resolved
// Shoot's status.technicalID.
//
// technicalID must come from the Shoot object, never from a tool argument and
// never recomputed from project and shoot names (ADR-0003).
func (p *Policy) AuthorizeSeedRead(technicalID, namespace string) error {
	if !p.SeedAccessAllowed() {
		return fmt.Errorf("%w: persona %q has no seed access", ErrPersonaDenied, p.persona)
	}
	if technicalID == "" {
		return fmt.Errorf("%w: empty technicalID", ErrNamespaceNotPinned)
	}
	if namespace != technicalID {
		return fmt.Errorf("%w: attempted %q, resolved %q", ErrNamespaceNotPinned, namespace, technicalID)
	}
	return nil
}

// dns1123Label matches a Kubernetes DNS-1123 label, which is the shape of
// Gardener project and Shoot names.
var dns1123Label = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// maxIdentifierLen is the DNS-1123 label limit.
const maxIdentifierLen = 63

// ValidateIdentifier rejects anything that is not a plain DNS-1123 label.
//
// This matters beyond tidiness: validated identifiers are the only cluster-
// derived strings allowed into server-authored text (ADR-0004). Everything that
// passes here is safe to interpolate into a finding title; everything else has
// to stay in evidence[].value.
func ValidateIdentifier(kind, value string) error {
	switch {
	case value == "":
		return fmt.Errorf("%w: %s is empty", ErrInvalidIdentifier, kind)
	case len(value) > maxIdentifierLen:
		return fmt.Errorf("%w: %s is %d characters, limit %d",
			ErrInvalidIdentifier, kind, len(value), maxIdentifierLen)
	case !dns1123Label.MatchString(value):
		// The offending value is deliberately not echoed back verbatim beyond
		// quoting: it is attacker-controlled, and this message is
		// server-authored text.
		return fmt.Errorf("%w: %s %q is not a DNS-1123 label", ErrInvalidIdentifier, kind, sanitizeForMessage(value))
	}
	return nil
}

// sanitizeForMessage reduces a rejected value to something safe to name in an
// error: printable ASCII only, and short. A rejected identifier still ends up
// in logs and error responses, so it must not smuggle control characters or
// instructions there.
func sanitizeForMessage(s string) string {
	const limit = 32

	var b strings.Builder
	for i, r := range s {
		if i >= limit {
			b.WriteString("...")
			break
		}
		if r < 0x20 || r > 0x7e {
			b.WriteByte('?')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
