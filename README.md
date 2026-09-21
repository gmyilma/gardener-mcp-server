# gardener-mcp-server

**Gardener-aware diagnostics for AI assistants, over the Model Context Protocol.**

[![ci](https://github.com/gmyilma/gardener-mcp-server/actions/workflows/ci.yaml/badge.svg)](https://github.com/gmyilma/gardener-mcp-server/actions/workflows/ci.yaml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)
[![REUSE status](https://api.reuse.software/badge/github.com/gmyilma/gardener-mcp-server)](https://api.reuse.software/info/github.com/gmyilma/gardener-mcp-server)

> [!WARNING]
> **Experimental.** This project is at milestone M0 (repository bootstrap). No
> MCP tools are implemented yet. Interfaces, schemas and rule IDs will change
> without notice before v0.1.0.

`gardener-mcp-server` gives MCP-compatible assistants and agents a safe,
structured view of Gardener's operational model. Given a Shoot, it resolves the
Garden → Seed → Shoot topology, collects the relevant state (Shoot status,
extension resources, ManagedResources, machines, etcd, nodes) with
least-privilege read-only credentials, and returns curated findings with
evidence — so an assistant can explain *why* a cluster is unhealthy without
direct cluster access.

It does not change your clusters.

## Why not a generic Kubernetes MCP server?

Diagnosing an unhealthy Shoot spans three API servers and three credential
classes, and needs knowledge specific to Gardener. Generic Kubernetes MCP
servers work against one kubeconfig at a time and offer object-level verbs, so
the model has to rediscover the topology itself — which requires broad
credentials, burns tokens, and fails unpredictably.

This server does the traversal deterministically and hands back interpreted
findings instead of raw objects.

## Architecture

```text
MCP client / agent ──MCP──▶ gardener-mcp-server ──read-only──▶ Garden · Seed · Shoot
```

The server contains no LLM. Topology resolution, evidence collection,
sanitization and rule evaluation are all deterministic; natural-language
reasoning stays in the calling agent.

See [docs/design/architecture-review.md](docs/design/architecture-review.md) for
the full design, threat model and RBAC.

## Design principles

- **Gardener semantics, not kubectl over MCP.** No arbitrary `get`/`list`, no
  shell, no generic object access.
- **No resource-mutation path.** The server holds no write credentials.
- **Least privilege.** Viewer kubeconfigs for Shoots; Seed access for operators
  only, pinned to a single control-plane namespace.
- **Cluster data is untrusted.** Free text from conditions, events and provider
  messages is labelled as such and never mixed into server-authored conclusions.
- **Deterministic rules with stable IDs.** Output is reproducible and testable.
- **Vendor-neutral.** Works with any MCP client; no dependency on a specific
  agent runtime.

## Deployment modes

| Mode | Transport | Whose identity reaches Gardener | Use it for |
|---|---|---|---|
| **local** | stdio | Your own Garden kubeconfig | Shoot owners and individual operators |
| **service** | streamable HTTP | The server's ServiceAccount, single-tenant | A shared team or operator assistant |

Service mode is single-tenant by design: one persona and one project scope per
deployment. Everyone who can reach the endpoint gets that deployment's full
view, so the endpoint must be restricted. Caller tokens are never forwarded to
Kubernetes.

## Scope (planned for v0.1)

`list_shoots` · `get_shoot_status` · `diagnose_shoot` · `inspect_control_plane` ·
`inspect_workers` · `inspect_etcd` · `list_shoot_events` · `check_shoot_versions`

## Non-goals

- Autonomous remediation, or replacing Gardener operators.
- Generic Kubernetes access, logs, or shell.
- Bundling an LLM or an agent framework.

## Getting started

Requires [mise](https://mise.jdx.dev), which pins the Go toolchain and every dev
tool for this repository.

```console
git clone https://github.com/gmyilma/gardener-mcp-server.git
cd gardener-mcp-server
mise install      # Go 1.26.8 + pinned tools, from mise.lock
make tools        # govulncheck into ./bin
make verify       # lint, tidy check, vulncheck, reuse, tests
```

There is no virtualenv to activate. `go.mod` already isolates dependencies per
project, and `mise` selects the toolchain when you enter the directory. See
[CONTRIBUTING.md](CONTRIBUTING.md).

## Roadmap

| Version | Content |
|---|---|
| v0.1 | Read-only diagnostics across Garden, Seed and Shoot |
| v0.2 | Remediation *proposals* — advice only, with server-computed impact |
| v0.3 | Opt-in GitOps pull requests |
| later | Human-approved remediation, only if the community needs it |

## Security

The server has no resource-mutation path and never emits credentials. To report
a vulnerability, see [SECURITY.md](SECURITY.md) — please do not open a public
issue.

## Licence

Apache-2.0 for code, CC-BY-4.0 for documentation. See [LICENSE](LICENSE) and
[REUSE.toml](REUSE.toml).

This is an independent project. It is not an official Gardener project and is
not affiliated with or endorsed by the Gardener maintainers.
