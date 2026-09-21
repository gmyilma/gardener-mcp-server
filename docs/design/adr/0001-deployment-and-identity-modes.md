# ADR-0001: Deployment and identity modes

**Status:** Accepted
**Date:** 2026-09-21

## Context

An MCP server that reads Gardener state has to answer a question that "the agent
is read-only" does not address: **on whose authority does a read happen?**

A single shared in-cluster server running under an operator ServiceAccount is a
classic confused deputy. Every caller who can reach the endpoint inherits the
server's view — which, for an operator identity, is the whole fleet. In v0.1 the
dominant risk is therefore **information disclosure**, not mutation, and
read-only RBAC does nothing about it.

Passing the caller's token through to Kubernetes would solve attribution, but
the MCP authorization specification forbids token pass-through, and
impersonation rights are powerful enough to be their own liability.

## Decision

Ship **two** deployment modes, and make the identity question explicit in each.

**Local mode** uses the stdio transport and the *user's own* Garden kubeconfig
(gardenlogin/OIDC). Gardener's RBAC decides what is visible, exactly as it does
for `kubectl`. There is no confused deputy because there is no shared identity.
This is the mode for Shoot owners and individual operators.

**Service mode** uses streamable HTTP and the server's own ServiceAccount, and
is **single-tenant by construction**: one persona (`--persona`) and one project
allowlist (`--projects`) per deployment. Everyone who can reach the endpoint
gets that deployment's full view. This is a documented property, stated in
user-facing docs and not only in code, so that operators scope the endpoint
deliberately. Caller authentication is required (mTLS or a bearer token, plus a
NetworkPolicy).

Caller tokens are **never** forwarded to Kubernetes.

Multi-tenant mode — per-caller identity via token exchange or impersonation — is
deferred. It is the honest way to serve many users from one deployment, and it
is not worth its complexity before anyone needs it.

## Consequences

A team wanting one shared endpoint for two projects with different audiences
must run two deployments. That is more operational work, and it is the point:
the blast radius of a deployment equals its configured scope.

Persona becomes a startup flag rather than a request attribute, so it can be
validated once and cannot be influenced by tool arguments. The policy layer
still re-checks it per call.

The operator persona in service mode remains genuinely powerful: it can read
every Shoot's control plane on a Seed. Restricting network access to that
endpoint is a deployment responsibility this project cannot enforce.

Revisit when a concrete user needs per-caller identity from one deployment, and
treat that as a design exercise, not a configuration flag.

## Alternatives considered

**One shared server with operator credentials, relying on "read-only".** The
original draft. Rejected: it makes every caller an operator and mistakes
mutation for the main risk.

**Token pass-through.** Forwarding the caller's MCP token to the Kubernetes API.
Rejected: forbidden by the MCP authorization spec, and it couples the server to
the identity provider topology of every landscape.

**Kubernetes impersonation.** The server holds impersonation rights and acts as
the caller. Rejected for v0.1: correct in principle, but impersonation rights
are close to cluster-admin in effect, and getting the audit story right is more
work than the first users justify. This is the natural basis for a future
multi-tenant mode.
