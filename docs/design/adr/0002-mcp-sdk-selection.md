# ADR-0002: Use the official MCP Go SDK

**Status:** Accepted
**Date:** 2026-09-21

## Context

Two mature Go MCP implementations exist: `modelcontextprotocol/go-sdk` (v1.8.0,
official, under the `modelcontextprotocol` organisation) and `mark3labs/mcp-go`
(v1.1.0, community-maintained, historically wider adoption in existing Go MCP
servers).

Two project properties bear on the choice. First, the design depends on **strict
input and output schemas** — tool arguments are the untrusted boundary
(threat model, TB1), and a schema that drifts from the Go types is a schema that
will eventually be wrong. Second, the project hopes to be adopted by, or donated
to, a CNCF/NeoNephos-adjacent community, where neutral governance matters.

The official SDK's `mcp.AddTool[In, Out]` infers both schemas from Go types via
`google/jsonschema-go` and validates against them. It tracks spec revision
2026-07-28, including the stateless model that allows horizontal scaling in
service mode. It has 8 direct dependencies and ships a conformance suite.

## Decision

Use `github.com/modelcontextprotocol/go-sdk`.

Keep it behind a small internal `tools.Registry` interface in `internal/server`,
so that tool packages depend on our abstraction rather than on the SDK directly.

## Consequences

Schemas derive from the Go types that handlers already use, so the type and the
contract cannot drift apart. Tightening a schema beyond what inference produces
— enums for `focus`, DNS-1123 patterns for `project` and `shoot`, `maxLength` —
is done in code starting from `jsonschema.For[T](nil)`, then committed to
`schemas/` for contract tests.

Handlers still validate their own input. The SDK validating against the schema
is a convenience, not a security control: relying on the client having honoured
the schema is exactly the assumption TB1 says not to make.

The registry interface costs an indirection that will look redundant while there
is only one SDK. That is accepted as the price of making a switch cheap.

MCP sampling is deprecated as of 2026-07-28, which is convenient: the server
makes no LLM calls anyway (ADR-0004).

## Alternatives considered

**`mark3labs/mcp-go`.** Good library with a builder-style schema API, input and
output validation, `mcptest`, OpenTelemetry and hooks. Rejected on two counts:
schemas are built rather than inferred, so they can drift from the Go types; and
it depends on a small maintainer group, which is a weaker footing for a project
aiming at community donation. Its wider existing adoption does not outweigh
these.

**Implementing the protocol directly.** Rejected: MCP is still moving, and
tracking spec revisions by hand is work that produces nothing specific to
Gardener, which is where the value of this project actually lies.
