# ADR-0005: No resource-mutation path

**Status:** Accepted
**Date:** 2026-09-21

## Context

The long-term vision includes remediation. The temptation is to build a small
piece of it early — a `RemediationProposal` CRD, or a tool that sets the
`gardener.cloud/operation` annotation — because it demos well.

Three findings argue against that in v0.1 and v0.2.

**"Safe" operations are not uniformly safe.** `reconcile` rolls out pending spec
changes when `metadata.generation != status.observedGeneration`, which can mean
a node roll. `maintain` runs maintenance *immediately*, which can apply
automatic Kubernetes patch and machine-image updates — also a node roll.
Classifying impact therefore requires reading live state, not matching on an
operation name.

**The annotation is not a simple enum.** Its value can hold several operations
separated by `;`, and a second annotation,
`maintenance.gardener.cloud/operation`, schedules an operation for the
maintenance window. An exact-match enum check is necessary, in more than one
place.

**A CRD needs a consumer.** A `RemediationProposal` is inert without an approval
flow *and an applier controller holding write access to Shoots*. That controller
would be the most dangerous component in the project, and it adds nothing to
diagnostics. Separately, RBAC cannot restrict *which fields* a patch touches, so
a safe applier needs admission control — and whether gardener-apiserver enforces
`ValidatingAdmissionPolicy` or admission webhooks for `core.gardener.cloud`
Shoots is still unverified.

## Decision

Through v0.2 the server has **no resource-mutation path**. It will not patch,
update or delete resources, create ordinary Kubernetes resources, or execute
commands in workloads. It holds no write credential, so this is enforced by the
absence of RBAC rather than by code that could be bypassed.

Describe the project as having **"no resource-mutation path"**, not as
"read-only". Requesting `shoots/viewerkubeconfig` uses the Kubernetes `create`
verb on a subresource, so "every verb is read-only" would be literally false,
and a security claim that is technically false is worse than a narrower true one.

In v0.2, remediation appears as a **`ProposalDocument` returned as tool output**:
a document a human applies through the Dashboard or a GitOps repository, with
impact computed by the server and `unknown` always forcing the elevated tier.
Output-only proposals capture most of the value with zero write credentials.

The `RemediationProposal` CRD and its applier are deferred to v0.4+, and only if
a landscape without GitOps actually asks. The admission-enforcement question
must be settled before that work starts.

## Consequences

The project cannot demo "the AI fixed it". It demos "the AI explained it,
correctly, with evidence" — which is the claim worth making and the one that is
much easier to get accepted upstream.

Security review becomes tractable: a reviewer can confirm the absence of write
verbs in the ClusterRoles and be largely done. A unit test asserts the
ClusterRole grants no write verb and never `shoots/adminkubeconfig`.

For GitOps-managed landscapes this may be permanent, and that is fine: a pull
request already *is* the proposal, branch protection is the approval, and git
history is the audit trail.

The impact classifier must be built before any proposal ships, including for
output-only documents. Advice that misrepresents its own blast radius is worse
than no advice, because it will be trusted.

Revisit v0.4 only when there is a named user, a landscape without GitOps, and a
resolved answer on admission enforcement.

## Alternatives considered

**Tools that set the operation annotation directly, gated by tool hints.**
Rejected: MCP tool hints are metadata for clients, not security controls, and
the confused-deputy analysis in ADR-0001 shows the caller is not a trustworthy
authorization input.

**The CRD now, applier later.** Rejected: an API with no consumer ossifies
around guesses. Building it after the approval flow is understood produces a
better API.

**GitOps pull requests in v0.1.** Attractive, since PR-only permissions are
genuinely limited. Deferred to v0.3 rather than rejected: it depends on
ownership detection being trustworthy first, and ownership defaults to `Unknown`
precisely because guessing is what causes wrong advice.
