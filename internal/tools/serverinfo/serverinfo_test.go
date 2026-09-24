// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0

package serverinfo_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gmyilma/gardener-mcp-server/internal/audit"
	"github.com/gmyilma/gardener-mcp-server/internal/policy"
	"github.com/gmyilma/gardener-mcp-server/internal/server"
	"github.com/gmyilma/gardener-mcp-server/internal/tools/serverinfo"
)

// connect builds a server with server_info registered and returns a connected
// client session. Exercising the tool over the real protocol, rather than
// calling the handler directly, is what proves the schema inference works.
func connect(t *testing.T, cfg serverinfo.Config) *mcp.ClientSession {
	t.Helper()

	pol, err := policy.New(policy.Config{Persona: policy.Persona(cfg.Persona)})
	if err != nil {
		t.Fatalf("policy.New() = %v", err)
	}

	srv, err := server.New(server.Options{
		Version: cfg.Version,
		Policy:  pol,
		Audit:   audit.New(&bytes.Buffer{}),
	})
	if err != nil {
		t.Fatalf("server.New() = %v", err)
	}
	serverinfo.Register(srv, cfg)

	clientT, serverT := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	srvErr := make(chan error, 1)
	go func() { srvErr <- srv.ServeTransport(ctx, serverT) }()

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).
		Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("connect = %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Logf("close: %v", err)
		}
	})

	return session
}

func TestServerInfoReportsDeployment(t *testing.T) {
	t.Parallel()

	session := connect(t, serverinfo.Config{
		Version:  "v0.0.1-test",
		Persona:  "operator",
		Mode:     "local",
		Projects: []string{"local"},
		Tools:    []string{"server_info"},
	})

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "server_info",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() = %v", err)
	}
	if res.IsError {
		t.Fatalf("server_info returned an error: %+v", res.Content)
	}

	got, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent is %T, want an object", res.StructuredContent)
	}

	want := map[string]any{
		"name":    "gardener-mcp-server",
		"version": "v0.0.1-test",
		"persona": "operator",
		"mode":    "local",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}

	// The no-mutation guarantee is stated explicitly so a caller does not have
	// to infer it, and it must never be reported as false.
	if got["noResourceMutationPath"] != true {
		t.Errorf("noResourceMutationPath = %v, want true", got["noResourceMutationPath"])
	}
}

// TestInputSchemaRejectsArguments checks that the empty Input struct produces a
// closed schema. additionalProperties:false is what review §8 requires against
// schema bypass, and here it comes from the Go type rather than by hand.
func TestInputSchemaRejectsArguments(t *testing.T) {
	t.Parallel()

	session := connect(t, serverinfo.Config{Version: "v", Persona: "shoot-owner", Mode: "local"})

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() = %v", err)
	}

	var tool *mcp.Tool
	for _, tl := range res.Tools {
		if tl.Name == "server_info" {
			tool = tl
		}
	}
	if tool == nil {
		t.Fatal("server_info not listed")
	}
	if tool.InputSchema == nil {
		t.Fatal("server_info has no input schema")
	}

	// Sending an unexpected argument must be refused, not silently ignored.
	bad, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "server_info",
		Arguments: map[string]any{"unexpected": "value"},
	})
	if err == nil && !bad.IsError {
		t.Error("an unexpected argument was accepted; the schema is not closed")
	}
}

func TestEmptyProjectsIsOmitted(t *testing.T) {
	t.Parallel()

	session := connect(t, serverinfo.Config{Version: "v", Persona: "shoot-owner", Mode: "local"})

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "server_info",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() = %v", err)
	}

	got, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structuredContent is %T, want an object", res.StructuredContent)
	}

	// An absent allowlist must not appear as an empty list, which would read
	// as "no projects permitted" rather than "no restriction configured".
	if v, present := got["projects"]; present {
		t.Errorf("projects = %v, want the field omitted", v)
	}
}
