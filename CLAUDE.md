# gardener-mcp-server

An open-source Go MCP server exposing Gardener-aware diagnostics to
MCP-compatible assistants.

> Expose Gardener operational semantics as safe, structured MCP tools instead of
> giving an AI agent arbitrary kubectl or shell access.

The server contains no LLM. Topology resolution, evidence collection,
sanitization and rule evaluation are deterministic; natural-language reasoning
belongs to the calling agent.

---

## Current phase — UPDATE THIS AS MILESTONES LAND

**Building M1** (server skeleton). Nothing Gardener-specific is implemented yet.

| Milestone | State |
|---|---|
| M0 bootstrap | done |
| M1.1 envelope + sanitizer | done — `internal/envelope` |
| M1.2 policy layer | done — `internal/policy` |
| **M1.3 audit log** | **next** |
| M1.4 go-sdk server, stdio, one fake tool | |
| M1.5 schema generation into `schemas/` | |
| M1.6 streamable HTTP + caller auth | |
| M2+ | see the roadmap in the architecture review §14 |

**Do not implement features from later milestones.** The long-term architecture
describes an eight-tool v0.1; the current target is an experimental **v0.0.1
exposing one real tool, `diagnose_shoot(project, shoot)`**. Anything beyond the
approved scope is deferred.

---

## Design decisions

The five decisions that constrain everything else are recorded as ADRs. Read
these before changing anything structural:

@docs/design/adr/README.md

Full analysis, threat model, RBAC and tool catalogue — large, read the relevant
section rather than the whole file: `docs/design/architecture-review.md`

The MVP specification (`docs/design/mvp-v0.0.1.md`) may not exist yet. If it
does not, do not assume its contents.

---

## Invariants

These are not preferences. A change breaking one will not be merged, however
useful it looks.

1. **Gardener semantics, not kubectl over MCP.** No arbitrary `get`/`list`, no
   shell, no exec, no generic object access. Tools model Gardener concepts.
2. **No resource-mutation path.** No patch, update, delete, or create of
   ordinary resources. Described that way rather than "read-only", because
   requesting a viewer kubeconfig uses the `create` verb (ADR-0005).
3. **Credentials never become data.** No kubeconfig, token or certificate in any
   response, log, error, evidence item or test snapshot.
   `shoots/adminkubeconfig` is never requested.
4. **Cluster text is untrusted.** It belongs in `evidence[].value` with a trust
   label, and is never interpolated into server-authored fields (ADR-0004).
5. **Topology is read, not computed.** Always `Shoot.status.technicalID`; never
   reconstruct the Seed namespace from project and shoot names (ADR-0003).
6. **Seed reads are namespace-pinned** to the resolved `technicalID`, and the
   shoot-owner persona attempts no Seed call at all (ADR-0001).

---

## Commands

```console
mise install && make tools    # one-time setup
make verify                   # lint + tidy + vulncheck + reuse + tests — what CI runs
make test                     # unit tests, race detector, coverage
make lint                     # golangci-lint
make build                    # binary into ./bin
make help                     # everything else
```

## Toolchain

- **Go comes from mise, not the system.** `.mise.toml` pins Go and the dev tool
  binaries; `mise.lock` holds checksums. There is **no virtualenv** — Go does not
  need one, because `go.mod` already isolates dependencies per project.
- **Dev tools are not `go.mod` `tool` directives**, deliberately: that would pull
  their whole dependency trees into `go.sum` and defeat the dependency-weight
  constraint in review §7. `go.mod` lists only what ships in the binary.
- **`GOBIN` points at `./bin`**, so `go install` never touches `~/go/bin`.
- **`reuse` is the only Python tool** and is always run isolated via `uvx` or
  `pipx`. Never `pip install` into a shared interpreter.

---

## Go guidance

The project owner is experienced with Java and Python and is learning Go through
this project.

- Straightforward idiomatic Go. Small interfaces, defined next to their
  consumers, not next to their implementations.
- Explicit dependency wiring. No DI framework, no factories, no Java-style
  abstraction layers.
- Prefer the standard library. Avoid goroutines and channels until there is a
  demonstrated need.
- Keep functions small and testable. Table-driven tests for anything enumerable.

**When introducing a non-obvious Go idiom, briefly explain:** what it does, why
it is idiomatic Go, and the approximate Java or Python equivalent.

## Working style

- **Incremental.** Do not generate large amounts of code in one step unless
  asked. Each milestone lands as several small, independently green commits.
- `make verify` must pass before committing.
- Conventional Commits, signed off (`git commit -s`). SPDX header on every file.
- **No literal invisible characters in source.** Use `\uXXXX` escapes for
  zero-width, bidi and homoglyph characters in test fixtures — a codebase about
  the dangers of invisible text should not hide any.

## Development environment

A Gardener local setup for testing lives on a separate host; connection details
and the environment fixes it needed are in the agent's project memory.
