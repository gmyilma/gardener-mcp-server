# Contributing

Thanks for your interest. This project is experimental and pre-v0.1.0, so the
most valuable contributions right now are design feedback and real Gardener
failure modes worth writing diagnostic rules for.

Please open an issue before starting anything substantial. It is a small
project, and a short conversation saves rework.

## Development setup

The toolchain is pinned with [mise](https://mise.jdx.dev). Install it, then:

```console
git clone https://github.com/gmyilma/gardener-mcp-server.git
cd gardener-mcp-server
mise install      # Go + golangci-lint, goreleaser, syft, cosign, from mise.lock
make tools        # govulncheck into ./bin
make verify       # everything CI runs on a pull request
```

`mise install` reads exact versions and checksums from `mise.lock`, so you get
the same tools as CI and as every other contributor.

### A note for people coming from Python or Java

There is **no virtualenv to activate**, and you do not need one. Python needs
venvs because `site-packages` is a single flat namespace where two projects
fight over one version of a library. Go has no such namespace: the module cache
is content-addressed and keyed by version, so `go.mod` already gives complete
per-project dependency isolation with nothing to switch on or off.

What `mise` adds is the part `go.mod` does not cover:

| Leaks globally | Python equivalent | Handled by |
|---|---|---|
| The Go toolchain version | `.venv/bin/python` | `go`/`toolchain` in `go.mod`, plus `.mise.toml` |
| Dev tool binaries | `.venv/bin/` | `.mise.toml` + `GOBIN=./bin` |
| Shell `PATH` wiring | `activate` | `mise activate` in your shell rc |

Dev tools are deliberately **not** `go.mod` `tool` directives. That would pull
golangci-lint's and goreleaser's whole dependency trees into `go.sum` and
`go mod graph`, and keeping the runtime dependency surface small and auditable
is a design constraint here (architecture review, section 7).

## Make targets

```console
make help              # list everything
make build             # binary into ./bin
make test              # unit tests, race detector, coverage
make lint              # golangci-lint
make check             # lint + tidy + vulncheck + reuse
make verify            # check + test — matches CI exactly
make snapshot          # local release build, publishes nothing
```

## Design constraints

These are not style preferences. A change that breaks one of them will not be
merged, however useful it looks. Full reasoning is in
[docs/design/architecture-review.md](docs/design/architecture-review.md).

- **Gardener semantics, not kubectl over MCP.** No arbitrary `get`/`list`, no
  shell or exec, no generic object access. Tools model Gardener operational
  concepts.
- **No resource-mutation path.** No patch, update, delete or create of ordinary
  resources.
- **Credentials never become data.** No kubeconfig, token or certificate in any
  response, log, error, evidence item or test snapshot.
  `shoots/adminkubeconfig` is never requested.
- **Cluster text is untrusted.** It belongs in `evidence[].value` with a trust
  label, and is never interpolated into server-authored fields.
- **Topology is derived safely.** Always read `Shoot.status.technicalID`; never
  reconstruct the Seed namespace from project and Shoot names.
- **No LLM in the server.** Rules are deterministic and carry stable IDs.

## Go style

Straightforward, idiomatic Go. Small interfaces, defined next to their
consumers. Explicit dependency wiring rather than a DI framework. No factories
or abstraction layers carried over from Java. Prefer the standard library. Avoid
goroutines and channels until there is a demonstrated need.

If you introduce a non-obvious Go idiom, add a short comment saying what it does
and why it is idiomatic — the maintainer is learning Go through this project and
review goes faster that way.

## Tests

- Table-driven tests for anything with enumerable cases.
- Golden-file tests for diagnostic output, so rule changes show up as diffs.
- Every new diagnostic rule needs a fixture and a page under `docs/rules/`.
- Sanitizer changes need cases for control characters, bidi characters,
  oversized values and the prompt-injection corpus.
- Any code touching credentials needs a test asserting that no output contains
  `certificate-authority-data` or `token:`.

## Commits and pull requests

[Conventional Commits](https://www.conventionalcommits.org): `feat:`, `fix:`,
`docs:`, `test:`, `ci:`, `build:`, `chore:`, `refactor:`. The release changelog
is generated from these, so the prefix matters.

Write the body for someone reading it in a year with no context: what changed
and why, not how. Reference the architecture review section when a change
implements or alters a documented decision.

Keep pull requests small and focused. `make verify` must pass before you open
one.

### Sign-off

Commits must carry a `Signed-off-by` line certifying the
[Developer Certificate of Origin](https://developercertificate.org):

```console
git commit -s -m "feat: add ..."
```

## Licensing

Apache-2.0 for code, CC-BY-4.0 for documentation. Every file needs an SPDX
header:

```go
// SPDX-FileCopyrightText: 2026 Girma Yilma
//
// SPDX-License-Identifier: Apache-2.0
```

`make reuse-lint` checks this, and so does CI.

## Code of conduct

Participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
