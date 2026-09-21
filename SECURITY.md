# Security policy

## Reporting a vulnerability

**Please do not open a public issue for a security problem.**

Report privately, whichever you prefer:

- [GitHub private vulnerability reporting](https://github.com/gmyilma/gardener-mcp-server/security/advisories/new)
  (preferred — it keeps the report, the fix and the advisory together)
- Email **girmamamuye@gmail.com** with `[SECURITY]` in the subject

Please include the affected version or commit, what an attacker gains, and the
smallest reproduction you have. Redact any real cluster data: hostnames, project
names and tokens are not needed to demonstrate a bug.

**Never include a kubeconfig, token or Secret in a report.** If you believe a
credential has leaked through this software, rotate it first, then report.

### What to expect

| Stage | Target |
|---|---|
| Acknowledgement | 3 working days |
| Initial assessment | 10 working days |
| Fix or mitigation plan | agreed with you, based on severity |

This is currently a single-maintainer project, so these are honest targets
rather than a contractual SLA. Coordinated disclosure is welcome; credit is
given unless you prefer otherwise.

## Supported versions

The project is pre-v0.1.0 and experimental. Only `main` receives fixes. Once
v0.1.0 ships, this section will name a supported version window.

## Security model in brief

The full threat model is in
[docs/design/architecture-review.md](docs/design/architecture-review.md)
(sections 3 and 4). The properties that matter most:

- **No resource-mutation path.** The server holds no write credentials. It
  cannot patch, update, delete or exec. Note that requesting a viewer kubeconfig
  technically uses the Kubernetes `create` verb on a subresource, which is why
  the claim is "no resource-mutation path" rather than "every verb is read-only".
- **Credentials never become data.** Kubeconfigs, tokens and certificates must
  never appear in MCP responses, logs, errors, evidence or test snapshots. The
  credential broker returns clients, never bytes. `shoots/adminkubeconfig` is
  never requested.
- **Cluster data is untrusted.** Condition messages, `lastError` text, events,
  labels and annotations are attacker-controllable by anyone who can write to a
  cluster. They appear only as labelled evidence values and are never
  interpolated into server-authored conclusions.
- **Single-tenant service mode.** A service-mode deployment exposes its whole
  view to everyone who can reach the endpoint. That is a deployment decision,
  not a bug — restrict the endpoint. Caller tokens are never forwarded to
  Kubernetes.
- **Seed reads are namespace-pinned** to the resolved Shoot's
  `status.technicalID`, which is what prevents cross-Shoot disclosure.

### Findings we are especially interested in

- Any path where a credential reaches output, a log or an error.
- Any way to read outside the resolved Shoot's namespace or the deployment's
  project allowlist.
- Prompt injection that escapes the evidence envelope into server-authored text.
- Anything that causes a write against a Garden, Seed or Shoot cluster.

### Out of scope

- An LLM being *misled in its explanation* by untrusted cluster text that was
  correctly labelled as untrusted. That residual risk is documented and
  accepted; AI conclusions are advisory.
- A service-mode operator deployment showing fleet data to a caller who was
  granted access to that endpoint. That is the documented tenancy model.
- Vulnerabilities in Gardener, Kubernetes or other upstream projects — please
  report those to the relevant project.

## Release integrity

Releases ship checksums, an SBOM per artifact, and a keyless
[cosign](https://github.com/sigstore/cosign) signature recorded in the public
Rekor transparency log. Verification instructions are in each release's notes.

Dev tool versions are pinned with checksums in `mise.lock`, and CI runs
`govulncheck` on every pull request.
