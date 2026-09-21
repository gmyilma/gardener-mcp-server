// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package envelope

import "time"

// SchemaVersion identifies the response shape. Bump it when a change would
// break a consumer that parses the output.
const SchemaVersion = "gardener-mcp.diagnose/v1alpha1"

// Trust says where a value came from, and therefore how much weight a reader
// may put on it. This is the central distinction of ADR-0004.
//
// Go note: a named string type with typed constants is Go's idiom for an enum.
// It is weaker than a Java enum — any string can be converted to it — which is
// why values arriving from outside are validated rather than assumed.
type Trust string

const (
	// TrustServer marks a value this server generated: derived identifiers,
	// timestamps, rule output.
	TrustServer Trust = "trusted"

	// TrustEnum marks a value checked against a known Gardener enum or error
	// code list. It came from a cluster, but only recognised values survive.
	TrustEnum Trust = "trusted-enum"

	// TrustUntrusted marks free text originating in a cluster: condition
	// messages, lastError descriptions, events, provider messages.
	//
	// Anyone able to write to a cluster can choose this text. It must never be
	// interpolated into a server-authored field.
	TrustUntrusted Trust = "untrusted"
)

// Severity ranks a finding. The envelope's overall health is the worst
// severity present.
type Severity string

// Severity levels, worst first.
const (
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Confidence is declared by the rule that produced a finding, never computed.
// ADR-0004 explains why this is an enum and not a number.
type Confidence string

const (
	// ConfidenceHigh means a direct Gardener error code or condition reason.
	ConfidenceHigh Confidence = "high"
	// ConfidenceMedium means a correlation across layers.
	ConfidenceMedium Confidence = "medium"
	// ConfidenceLow means a heuristic, such as an age threshold.
	ConfidenceLow Confidence = "low"
)

// SourceStatus is the outcome of one collector's attempt to read one cluster.
// Recording failures explicitly is what lets a partial answer stay honest.
type SourceStatus string

// Collector outcomes. Anything other than SourceOK makes the result partial.
const (
	SourceOK        SourceStatus = "ok"
	SourceForbidden SourceStatus = "forbidden"
	SourceNotFound  SourceStatus = "notFound"
	SourceTimeout   SourceStatus = "timeout"
	SourceSkipped   SourceStatus = "skipped"
	SourceError     SourceStatus = "error"
)

// Completeness says whether every planned collector reported successfully.
type Completeness string

// Completeness values.
const (
	CompletenessComplete Completeness = "complete"
	CompletenessPartial  Completeness = "partial"
)

// Cluster names one of the three clusters a diagnosis may visit.
type Cluster string

// The three clusters a diagnosis may visit, in traversal order.
const (
	ClusterGarden Cluster = "garden"
	ClusterSeed   Cluster = "seed"
	ClusterShoot  Cluster = "shoot"
)

// Envelope is the single response shape every tool returns.
type Envelope struct {
	SchemaVersion string  `json:"schemaVersion"`
	Meta          Meta    `json:"meta"`
	Target        Target  `json:"target"`
	Summary       Summary `json:"summary"`

	// Findings are server-authored conclusions. Every string in here is a
	// template plus validated identifiers. No cluster text, ever.
	Findings []Finding `json:"findings"`

	// Evidence is top-level and referenced by ID, so a value observed once can
	// support several findings without being duplicated.
	Evidence []Evidence `json:"evidence"`

	// Sources records every API call attempted, including the ones that were
	// skipped or failed.
	Sources []Source `json:"sources"`
}

// Meta describes the call itself.
type Meta struct {
	Tool          string       `json:"tool"`
	ServerVersion string       `json:"serverVersion"`
	Persona       string       `json:"persona"`
	GeneratedAt   time.Time    `json:"generatedAt"`
	DurationMs    int64        `json:"durationMs"`
	Completeness  Completeness `json:"completeness"`
}

// Target identifies what was diagnosed. Every field here is server-derived or
// validated, so the whole struct is trusted.
type Target struct {
	Project   string `json:"project"`
	Shoot     string `json:"shoot"`
	Namespace string `json:"namespace"`

	// TechnicalID is always read from Shoot.status.technicalID and never
	// reconstructed from project and shoot names (ADR-0003, review section 5).
	TechnicalID string `json:"technicalID,omitempty"`

	// Seed carries both the spec and status seed names. When they differ, a
	// control-plane migration is in progress.
	Seed SeedRef `json:"seed"`

	Generation         int64 `json:"generation,omitempty"`
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// SeedRef holds the desired and actual Seed. status is written only after a
// successful reconcile, so the two differ during a migration.
type SeedRef struct {
	Spec   string `json:"spec,omitempty"`
	Status string `json:"status,omitempty"`
}

// Summary is the at-a-glance verdict.
type Summary struct {
	// Health is the worst severity among findings, or "healthy" if there are
	// none.
	Health string `json:"health"`
	// TopFindings lists finding IDs in the order a reader should consider them.
	TopFindings []string `json:"topFindings"`
}

// Finding is a server-authored conclusion tied to a deterministic rule.
type Finding struct {
	ID string `json:"id"`

	// RuleID is stable across releases and has a page under docs/rules/.
	// It is what makes a finding citable and testable.
	RuleID string `json:"ruleId"`

	Severity   Severity   `json:"severity"`
	Confidence Confidence `json:"confidence"`
	Component  string     `json:"component,omitempty"`
	WorkerPool string     `json:"workerPool,omitempty"`

	// Title and Interpretation are written by the server from templates and
	// validated identifiers. Putting cluster text in either would defeat the
	// whole trust model.
	Title          string `json:"title"`
	Interpretation string `json:"interpretation"`

	EvidenceRefs []string `json:"evidenceRefs,omitempty"`

	// NextSteps are diagnostic, never remediating. Through v0.2 this server
	// proposes no mutation (ADR-0005).
	NextSteps []NextStep `json:"nextSteps,omitempty"`

	Docs []string `json:"docs,omitempty"`
}

// NextStep suggests another tool call that would narrow the diagnosis.
type NextStep struct {
	Tool   string            `json:"tool"`
	Args   map[string]string `json:"args"`
	Reason string            `json:"reason"`
}

// Evidence is one observed value, with enough identity to be checked by hand.
type Evidence struct {
	ID string `json:"id"`

	Cluster     Cluster `json:"cluster"`
	ClusterName string  `json:"clusterName,omitempty"`

	// Credential names which identity read this, so an auditor can tell
	// operator-scoped reads from the caller's own.
	Credential string `json:"credential,omitempty"`

	Object     ObjectRef `json:"object"`
	Field      string    `json:"field"`
	ObservedAt time.Time `json:"observedAt"`

	Trust Trust `json:"trust"`

	// Value is the only place untrusted cluster text may appear.
	Value any `json:"value"`

	// Sanitized reports that control, bidi or zero-width characters were
	// removed. A reader seeing this knows the text was crafted, not merely odd.
	Sanitized bool `json:"sanitized,omitempty"`

	Truncated      bool `json:"truncated,omitempty"`
	OriginalLength int  `json:"originalLength,omitempty"`
}

// ObjectRef identifies the Kubernetes object a value came from.
type ObjectRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

// Source records one collector's attempt against one cluster.
type Source struct {
	Cluster     Cluster      `json:"cluster"`
	ClusterName string       `json:"clusterName,omitempty"`
	Credential  string       `json:"credential,omitempty"`
	Collector   string       `json:"collector"`
	Status      SourceStatus `json:"status"`
	DurationMs  int64        `json:"durationMs"`

	// Reason explains a non-ok status in server-authored words, for example
	// "persona" for a skipped Seed hop. It never carries an upstream error
	// string, which could itself be attacker-influenced.
	Reason string `json:"reason,omitempty"`
}
