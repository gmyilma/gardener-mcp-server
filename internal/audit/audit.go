// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Outcome is how a tool call ended. It is a closed set, so an audit reader can
// aggregate outcomes without parsing prose.
type Outcome string

// The outcomes a tool call can have.
const (
	// OutcomeOK means the call completed and returned a result. The result may
	// still be partial; see Event.Completeness.
	OutcomeOK Outcome = "ok"
	// OutcomeDenied means the policy layer refused the call before any cluster
	// was contacted.
	OutcomeDenied Outcome = "denied"
	// OutcomeInvalid means the arguments failed validation.
	OutcomeInvalid Outcome = "invalid"
	// OutcomeTimeout means the call exceeded its deadline.
	OutcomeTimeout Outcome = "timeout"
	// OutcomeError means an unexpected failure.
	OutcomeError Outcome = "error"
)

// Event is one auditable tool call.
//
// Every field is either a validated identifier, a closed enum, or a number.
// There is deliberately **no** free-form map and no field for raw cluster text:
// the strongest guarantee that a credential never reaches the audit log is that
// there is nowhere to put one (architecture-review.md §4, "kubeconfig leakage").
//
// Go note: a struct of named fields rather than a map[string]any is the shape
// that makes this reviewable. In Java you might reach for a Map<String,Object>
// and rely on discipline; here the compiler enforces it.
type Event struct {
	// Tool is the MCP tool name, e.g. "diagnose_shoot".
	Tool string

	// Caller identifies the authenticated caller in service mode. In local mode
	// the caller is the user running the process, and this stays empty.
	Caller string

	// Persona is the deployment's fixed persona.
	Persona string

	// Project and Shoot are the tool's target. Both have passed
	// policy.ValidateIdentifier before reaching here.
	Project string
	Shoot   string

	// TechnicalID is the resolved Seed namespace, recorded so an auditor can
	// confirm which control plane was read.
	TechnicalID string

	// Clusters lists which clusters were actually contacted, e.g.
	// ["garden", "seed", "shoot"]. A skipped layer does not appear.
	Clusters []string

	// Credentials names which identities were used, e.g.
	// ["garden-reader", "seed-reader"]. These are *names*, never material.
	Credentials []string

	Outcome Outcome

	// Completeness mirrors the envelope's meta.completeness.
	Completeness string

	// Findings is the number of findings returned, not their content.
	Findings int

	// Reason explains a non-ok outcome in server-authored words, such as
	// "project not allowed". It must never carry an upstream error string,
	// which could itself be attacker-influenced.
	Reason string

	Duration time.Duration
}

// Logger writes audit events.
//
// It wraps log/slog rather than defining its own format: structured logging is
// a solved problem in the standard library, and an operator can point it at
// whatever collector they already run.
type Logger struct {
	log *slog.Logger
}

// New returns a Logger writing JSON lines to w.
func New(w io.Writer) *Logger {
	return &Logger{
		log: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})).With(slog.String("kind", "audit")),
	}
}

// NewFromSlog wraps an existing slog.Logger, for embedding in a host
// application that already configured one.
func NewFromSlog(l *slog.Logger) *Logger {
	return &Logger{log: l.With(slog.String("kind", "audit"))}
}

// Record writes one event.
//
// Every string is scrubbed on the way out. That is redundant given the typed
// fields above, and deliberately so: this is the last point before data leaves
// the process, and the cost of a redundant check is far below the cost of the
// failure it guards against.
func (l *Logger) Record(ctx context.Context, e Event) {
	l.log.LogAttrs(ctx, slog.LevelInfo, "tool call",
		slog.String("tool", scrub(e.Tool)),
		slog.String("caller", scrub(e.Caller)),
		slog.String("persona", scrub(e.Persona)),
		slog.String("project", scrub(e.Project)),
		slog.String("shoot", scrub(e.Shoot)),
		slog.String("technicalID", scrub(e.TechnicalID)),
		slog.Any("clusters", scrubAll(e.Clusters)),
		slog.Any("credentials", scrubAll(e.Credentials)),
		slog.String("outcome", scrub(string(e.Outcome))),
		slog.String("completeness", scrub(e.Completeness)),
		slog.Int("findings", e.Findings),
		slog.String("reason", scrub(e.Reason)),
		slog.Int64("durationMs", e.Duration.Milliseconds()),
	)
}

// redacted replaces any value that looks like credential material.
const redacted = "[REDACTED]"

// credentialPatterns match things that must never appear in a log line.
//
// This is defence in depth, not the primary control. The primary control is
// that Event has no field for credential material. If one of these ever fires
// in practice, something upstream is wrong and the test suite should grow a
// case for it.
var credentialPatterns = []*regexp.Regexp{
	// PEM blocks: private keys, certificates.
	regexp.MustCompile(`(?i)-{5}BEGIN [A-Z ]*(PRIVATE KEY|CERTIFICATE)`),
	// kubeconfig fields that carry material.
	regexp.MustCompile(`(?i)(certificate-authority-data|client-certificate-data|client-key-data)\s*:`),
	// Bearer tokens and generic token assignments.
	regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/-]{8,}`),
	regexp.MustCompile(`(?i)\btoken\s*[:=]\s*\S{8,}`),
	// JWTs, which is what a ServiceAccount token looks like.
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.`),
}

// maxFieldLen bounds a single logged value. An unbounded field turns the audit
// log into a denial-of-service vector against whatever collects it.
const maxFieldLen = 256

// scrub makes a single value safe to log: redacted if it looks like a
// credential, stripped of characters that could forge log structure, and
// bounded in length.
func scrub(s string) string {
	if s == "" {
		return ""
	}

	for _, re := range credentialPatterns {
		if re.MatchString(s) {
			return redacted
		}
	}

	var b strings.Builder
	b.Grow(len(s))

	for i, r := range s {
		if i >= maxFieldLen {
			b.WriteString("...")
			break
		}
		switch {
		case r == '\n' || r == '\r':
			// Newlines would let a value forge an extra log record. The JSON
			// handler escapes them, but a downstream consumer reading lines
			// might not, so they are removed rather than escaped.
			b.WriteByte(' ')
		case unicode.Is(unicode.Cf, r), unicode.IsControl(r):
			// Same reasoning as the evidence sanitizer: characters that let
			// rendered text differ from its bytes have no place in an audit
			// trail, which exists to be read by humans after an incident.
			continue
		default:
			b.WriteRune(r)
		}
	}

	return b.String()
}

// scrubAll applies scrub to a slice, returning a new slice so the caller's
// data is never modified.
func scrubAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, scrub(s))
	}
	return out
}
