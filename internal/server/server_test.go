// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gmyilma/gardener-mcp-server/internal/audit"
	"github.com/gmyilma/gardener-mcp-server/internal/policy"
	"github.com/gmyilma/gardener-mcp-server/internal/server"
)

// shootInput is a tool input naming a Shoot. It satisfies server.ShootTarget,
// so the server authorizes it before the handler runs.
type shootInput struct {
	Project string `json:"project"`
	Shoot   string `json:"shoot"`
}

func (in shootInput) Target() (project, shoot string) { return in.Project, in.Shoot }

type echoOutput struct {
	Seen string `json:"seen"`
}

// harness wires a server to an in-process client, which is how the SDK expects
// servers to be tested: real protocol, no subprocess.
type harness struct {
	t            *testing.T
	session      *mcp.ClientSession
	auditBuf     *bytes.Buffer
	handlerCalls int
}

func newHarness(t *testing.T, cfg policy.Config, register func(*server.Server, *int)) *harness {
	t.Helper()

	pol, err := policy.New(cfg)
	if err != nil {
		t.Fatalf("policy.New() = %v", err)
	}

	var buf bytes.Buffer
	srv, err := server.New(server.Options{
		Version: "test",
		Policy:  pol,
		Audit:   audit.New(&buf),
	})
	if err != nil {
		t.Fatalf("server.New() = %v", err)
	}

	h := &harness{t: t, auditBuf: &buf}
	if register != nil {
		register(srv, &h.handlerCalls)
	}

	clientT, serverT := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	// Run returns when the client disconnects. Send rather than discard, so
	// errcheck is satisfied and a genuine failure is still observable.
	srvErr := make(chan error, 1)
	go func() { srvErr <- srv.ServeTransport(ctx, serverT) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil).
		Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect = %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("session close: %v", err)
		}
	})

	h.session = session
	return h
}

// auditRecords decodes the audit lines written so far.
func (h *harness) auditRecords() []map[string]any {
	h.t.Helper()

	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(h.auditBuf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			h.t.Fatalf("audit line is not JSON: %v\n%s", err, line)
		}
		out = append(out, m)
	}
	return out
}

func registerEcho(srv *server.Server, calls *int) {
	server.Register(srv, server.ToolDef{
		Name:        "echo_shoot",
		Title:       "Echo",
		Description: "Test tool that echoes its target.",
	}, func(_ context.Context, in shootInput) (echoOutput, error) {
		*calls++
		return echoOutput{Seen: in.Project + "/" + in.Shoot}, nil
	})
}

func TestToolIsListed(t *testing.T) {
	t.Parallel()

	h := newHarness(t, policy.Config{Persona: policy.PersonaOperator}, registerEcho)

	res, err := h.session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() = %v", err)
	}

	var found *mcp.Tool
	for _, tool := range res.Tools {
		if tool.Name == "echo_shoot" {
			found = tool
		}
	}
	if found == nil {
		t.Fatalf("echo_shoot not listed; got %d tools", len(res.Tools))
	}

	// v0.1 tools are read-only and closed-world. These are hints for clients,
	// not security controls, but they should still be accurate.
	if found.Annotations == nil {
		t.Fatal("tool has no annotations")
	}
	if !found.Annotations.ReadOnlyHint {
		t.Error("readOnlyHint = false, want true")
	}
	if !found.Annotations.IdempotentHint {
		t.Error("idempotentHint = false, want true")
	}
	if found.InputSchema == nil {
		t.Error("tool has no input schema — schema inference did not run")
	}
}

func TestAuthorizedCallReachesHandler(t *testing.T) {
	t.Parallel()

	h := newHarness(t, policy.Config{
		Persona:  policy.PersonaOperator,
		Projects: []string{"allowed"},
	}, registerEcho)

	res, err := h.session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "echo_shoot",
		Arguments: shootInput{Project: "allowed", Shoot: "dev"},
	})
	if err != nil {
		t.Fatalf("CallTool() = %v", err)
	}
	if res.IsError {
		t.Fatalf("CallTool returned IsError; content: %+v", res.Content)
	}
	if h.handlerCalls != 1 {
		t.Errorf("handler called %d times, want 1", h.handlerCalls)
	}

	recs := h.auditRecords()
	if len(recs) != 1 {
		t.Fatalf("audit records = %d, want 1", len(recs))
	}
	if recs[0]["outcome"] != "ok" {
		t.Errorf("outcome = %v, want ok", recs[0]["outcome"])
	}
	if recs[0]["project"] != "allowed" {
		t.Errorf("project = %v, want allowed", recs[0]["project"])
	}
}

// TestDeniedProjectNeverReachesHandler is the important one: policy must gate
// the call before any handler code runs, not after.
func TestDeniedProjectNeverReachesHandler(t *testing.T) {
	t.Parallel()

	h := newHarness(t, policy.Config{
		Persona:  policy.PersonaOperator,
		Projects: []string{"allowed"},
	}, registerEcho)

	res, err := h.session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "echo_shoot",
		Arguments: shootInput{Project: "forbidden", Shoot: "dev"},
	})
	// A refused call surfaces as a tool error, not a transport error.
	if err == nil && !res.IsError {
		t.Fatal("call to a disallowed project succeeded")
	}

	if h.handlerCalls != 0 {
		t.Errorf("handler ran %d times for a denied call, want 0", h.handlerCalls)
	}

	recs := h.auditRecords()
	if len(recs) != 1 {
		t.Fatalf("audit records = %d, want 1", len(recs))
	}
	if recs[0]["outcome"] != "denied" {
		t.Errorf("outcome = %v, want denied", recs[0]["outcome"])
	}
	if recs[0]["reason"] != "project not allowed" {
		t.Errorf("reason = %v, want %q", recs[0]["reason"], "project not allowed")
	}
}

func TestInvalidIdentifierIsRejected(t *testing.T) {
	t.Parallel()

	h := newHarness(t, policy.Config{Persona: policy.PersonaOperator}, registerEcho)

	res, err := h.session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "echo_shoot",
		Arguments: shootInput{Project: "../../etc", Shoot: "dev"},
	})
	if err == nil && !res.IsError {
		t.Fatal("call with a path-traversal project succeeded")
	}
	if h.handlerCalls != 0 {
		t.Errorf("handler ran %d times, want 0", h.handlerCalls)
	}

	recs := h.auditRecords()
	if len(recs) != 1 || recs[0]["outcome"] != "invalid" {
		t.Errorf("audit = %+v, want a single record with outcome=invalid", recs)
	}
}

func TestNewRequiresPolicyAndAudit(t *testing.T) {
	t.Parallel()

	pol, err := policy.New(policy.Config{Persona: policy.PersonaShootOwner})
	if err != nil {
		t.Fatalf("policy.New() = %v", err)
	}

	if _, err := server.New(server.Options{Audit: audit.New(&bytes.Buffer{})}); err == nil {
		t.Error("New() without Policy succeeded, want an error")
	}
	if _, err := server.New(server.Options{Policy: pol}); err == nil {
		t.Error("New() without Audit succeeded, want an error")
	}
}
