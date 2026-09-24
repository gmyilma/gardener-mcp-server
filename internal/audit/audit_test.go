// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// record runs one event through a Logger and returns the decoded JSON line.
func record(t *testing.T, e Event) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	New(&buf).Record(context.Background(), e)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("audit line is not valid JSON: %v\nline: %s", err, buf.String())
	}
	return got
}

func TestRecordWritesExpectedFields(t *testing.T) {
	t.Parallel()

	got := record(t, Event{
		Tool:         "diagnose_shoot",
		Caller:       "system:serviceaccount:kagent:agent",
		Persona:      "operator",
		Project:      "local",
		Shoot:        "dev",
		TechnicalID:  "shoot--local--dev",
		Clusters:     []string{"garden", "seed"},
		Credentials:  []string{"garden-reader", "seed-reader"},
		Outcome:      OutcomeOK,
		Completeness: "partial",
		Findings:     3,
		Duration:     2140 * time.Millisecond,
	})

	want := map[string]any{
		"kind":         "audit",
		"tool":         "diagnose_shoot",
		"persona":      "operator",
		"project":      "local",
		"shoot":        "dev",
		"technicalID":  "shoot--local--dev",
		"outcome":      "ok",
		"completeness": "partial",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}

	if got["findings"] != float64(3) {
		t.Errorf("findings = %v, want 3", got["findings"])
	}
	if got["durationMs"] != float64(2140) {
		t.Errorf("durationMs = %v, want 2140", got["durationMs"])
	}
}

// TestCredentialMaterialIsRedacted is the test that matters. The architecture
// review lists kubeconfig leakage into logs as a distinct threat; this asserts
// the guard holds for every shape of credential we expect to see.
func TestCredentialMaterialIsRedacted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "PEM private key",
			value: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA\n-----END RSA PRIVATE KEY-----",
		},
		{
			name:  "PEM certificate",
			value: "-----BEGIN CERTIFICATE-----\nMIIDBTCCAe2gAwIBAgII\n-----END CERTIFICATE-----",
		},
		{
			name:  "kubeconfig CA data",
			value: "certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t",
		},
		{
			name:  "kubeconfig client cert",
			value: "client-certificate-data: LS0tLS1CRUdJTiBDRVJU",
		},
		{
			name:  "kubeconfig client key",
			value: "client-key-data: LS0tLS1CRUdJTiBSU0Eg",
		},
		{
			name:  "bearer token header",
			value: "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IjlwS201",
		},
		{
			name:  "token assignment",
			value: "token: abcdef0123456789",
		},
		{
			// A real ServiceAccount token, which is what shoots/viewerkubeconfig
			// returns and what must never be logged.
			name:  "JWT",
			value: "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJzeXN0ZW0ifQ.signature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Try it in every string field, since a leak could come from any.
			for _, field := range []string{"caller", "reason", "project", "technicalID"} {
				e := Event{Tool: "t", Outcome: OutcomeError}
				switch field {
				case "caller":
					e.Caller = tc.value
				case "reason":
					e.Reason = tc.value
				case "project":
					e.Project = tc.value
				case "technicalID":
					e.TechnicalID = tc.value
				}

				var buf bytes.Buffer
				New(&buf).Record(context.Background(), e)
				line := buf.String()

				if got := extractField(t, line, field); got != redacted {
					t.Errorf("field %s = %q, want %q", field, got, redacted)
				}

				// Belt and braces: no recognisable fragment survives anywhere.
				for _, frag := range []string{"BEGIN RSA", "BEGIN CERTIFICATE", "LS0tLS1", "eyJ"} {
					if strings.Contains(line, frag) {
						t.Errorf("fragment %q leaked into the log line: %s", frag, line)
					}
				}
			}
		})
	}
}

func extractField(t *testing.T, line, field string) string {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	s, ok := m[field].(string)
	if !ok {
		t.Fatalf("field %q is %T, want string", field, m[field])
	}
	return s
}

// TestNewlinesCannotForgeRecords checks that a value cannot inject what looks
// like a second log entry. The JSON handler escapes newlines, but a downstream
// consumer reading line-delimited output might not.
func TestNewlinesCannotForgeRecords(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	New(&buf).Record(context.Background(), Event{
		Tool:    "t",
		Reason:  "denied\n{\"kind\":\"audit\",\"outcome\":\"ok\"}",
		Outcome: OutcomeDenied,
	})

	if n := strings.Count(strings.TrimSpace(buf.String()), "\n"); n != 0 {
		t.Errorf("audit output spans %d extra lines, want a single record:\n%s", n, buf.String())
	}

	got := extractField(t, buf.String(), "reason")
	if strings.Contains(got, "\n") {
		t.Errorf("reason retained a newline: %q", got)
	}
}

func TestControlAndBidiCharactersAreStripped(t *testing.T) {
	t.Parallel()

	got := record(t, Event{
		Tool:    "t",
		Reason:  "clean\u202Ehsiugsid\u202C\u200Bmore\x07",
		Outcome: OutcomeError,
	})

	reason, ok := got["reason"].(string)
	if !ok {
		t.Fatalf("reason is %T, want string", got["reason"])
	}
	for _, bad := range []string{"\u202E", "\u202C", "\u200B", "\x07"} {
		if strings.Contains(reason, bad) {
			t.Errorf("reason retained %q: %q", bad, reason)
		}
	}
	if !strings.Contains(reason, "clean") {
		t.Errorf("legitimate text was destroyed: %q", reason)
	}
}

func TestLongValuesAreBounded(t *testing.T) {
	t.Parallel()

	got := record(t, Event{
		Tool:    "t",
		Reason:  strings.Repeat("x", 5000),
		Outcome: OutcomeError,
	})

	reason, ok := got["reason"].(string)
	if !ok {
		t.Fatalf("reason is %T, want string", got["reason"])
	}
	if len(reason) > maxFieldLen+len("...") {
		t.Errorf("reason is %d bytes, want <= %d", len(reason), maxFieldLen+3)
	}
}

func TestEmptyFieldsStayEmpty(t *testing.T) {
	t.Parallel()

	got := record(t, Event{Tool: "t", Outcome: OutcomeOK})

	if got["caller"] != "" {
		t.Errorf("caller = %v, want empty", got["caller"])
	}
	// A call that contacted nothing must not claim it contacted something.
	if c, ok := got["clusters"]; ok && c != nil {
		t.Errorf("clusters = %v, want null for an empty slice", c)
	}
}

// TestScrubDoesNotMutateCaller guards against the logger altering data the
// caller still holds.
func TestScrubDoesNotMutateCaller(t *testing.T) {
	t.Parallel()

	clusters := []string{"garden\n", "seed"}
	var buf bytes.Buffer
	New(&buf).Record(context.Background(), Event{
		Tool:     "t",
		Clusters: clusters,
		Outcome:  OutcomeOK,
	})

	if clusters[0] != "garden\n" {
		t.Errorf("caller's slice was modified: %q", clusters[0])
	}
}

func TestOutcomesAreDistinct(t *testing.T) {
	t.Parallel()

	seen := map[Outcome]bool{}
	for _, o := range []Outcome{OutcomeOK, OutcomeDenied, OutcomeInvalid, OutcomeTimeout, OutcomeError} {
		if o == "" {
			t.Error("an outcome constant is empty")
		}
		if seen[o] {
			t.Errorf("duplicate outcome value %q", o)
		}
		seen[o] = true
	}
}
