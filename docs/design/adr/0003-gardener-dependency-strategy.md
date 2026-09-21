# ADR-0003: Depend on Gardener's API module, not its root module

**Status:** Accepted
**Date:** 2026-09-21

## Context

The server needs typed access to Gardener resources (Shoot, Seed, Project,
CloudProfile, the `extensions.gardener.cloud` resources, ManagedResource), to
etcd-druid's `Etcd`, and to machine-controller-manager's Machine types.

Importing `github.com/gardener/gardener` at its root pulls the entire Gardener
dependency graph into a project whose job is to make a handful of read calls.
That is a large attack surface for the supply-chain risk in the threat model, a
large `go mod graph` to audit, and tight version coupling to a project that
releases often.

Verified against the upstream repositories: Gardener publishes its APIs as a
**separate module**, `github.com/gardener/gardener/pkg/apis` (tags
`pkg/apis/v1.151.x`), depending only on k8s api/apimachinery/component-base
plus semver and yaml. etcd-druid publishes `github.com/gardener/etcd-druid/api`.
MCM publishes **no** separate API module; its root module pins `k8s.io/apiserver`
and `code-generator`.

## Decision

Depend on `github.com/gardener/gardener/pkg/apis` and
`github.com/gardener/etcd-druid/api` for typed access.

For MCM, start with **minimal local types** in `internal/apis/mcm`, decoded from
unstructured objects and kept honest by contract tests against MCM's published
CRDs. Measure `go mod graph` in M1 and switch to the typed import if importing
MCM's root module turns out to be acceptable after all.

Use the controller-runtime **uncached** client (`client.New`) with a scheme
registering only these types. Diagnostics are request-scoped, so informer caches
on Seeds and Shoots would waste memory for no benefit. Use
`metav1.PartialObjectMetadataList` where only names and labels are needed.

Dev tools stay out of `go.mod`: they live in `.mise.toml` (see
[CONTRIBUTING.md](../../../CONTRIBUTING.md)), so `go.mod` describes only what
ships in the binary.

## Consequences

The dependency graph stays small enough to audit by hand, which is what makes
`govulncheck` output actionable rather than noise.

The project follows Gardener's **minor** versions. `pkg/apis` currently declares
`go 1.26.0` and k8s libraries `v0.36.x`, so the Go toolchain and the
controller-runtime version are both constrained by that choice; Renovate groups
them so they move together.

The MCM decision is the weak point. Local types are a duplicated contract, and
duplicated contracts rot. The contract tests are not optional, and this should
be revisited the moment they become burdensome or MCM publishes an API module.

Supporting a version window of N−2 Gardener minors means CI eventually runs
envtest against CRDs from each supported version. Optional features, such as
NamespacedCloudProfile, are detected through discovery rather than assumed.

## Alternatives considered

**Import `github.com/gardener/gardener` at the root.** Highest type safety,
simplest imports. Rejected: it brings the whole Gardener dependency graph and
couples the project tightly to Gardener's release cadence.

**Generated clientsets (`pkg/client/...`).** Rejected: they live in the root
module, so this inherits the previous option's cost. The controller-runtime
client covers the need.

**Unstructured or dynamic access throughout.** Minimal dependencies, loosest
coupling, no type safety. Rejected as a default, but adopted selectively for MCM
types and provider-specific `providerStatus`, where the schema is either
unavailable or genuinely provider-dependent.
