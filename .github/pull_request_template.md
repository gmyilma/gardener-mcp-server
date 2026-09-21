<!--
Thanks for contributing. Keep this short — the checklist matters more than prose.
-->

## What changed and why

<!-- What a reader in a year needs to know. Link the issue if there is one. -->

Fixes #

## Design impact

<!--
If this touches a documented decision, say which. If it changes one, this
probably needs an ADR rather than a pull request comment.
-->

- [ ] No change to a decision recorded in `docs/design/adr/`
- [ ] Changes a decision — new ADR included in this PR

## Invariants

These are the project's non-negotiables. Tick the ones your change touches, or
mark not applicable. See [CONTRIBUTING.md](../CONTRIBUTING.md).

- [ ] **No resource-mutation path.** No patch, update, delete, create of
      ordinary resources, or exec.
- [ ] **No credential can reach output.** Not in responses, logs, errors,
      evidence or test snapshots.
- [ ] **Untrusted cluster text stays in `evidence[].value`** and is not
      interpolated into server-authored fields.
- [ ] **Seed reads are pinned** to the resolved `status.technicalID`.
- [ ] **Topology is read, not computed** — `Shoot.status.technicalID`, never
      reconstructed from project and Shoot names.
- [ ] Not applicable to this change

## Tests

- [ ] `make verify` passes locally
- [ ] New or changed behaviour is covered by tests
- [ ] New diagnostic rule includes a fixture and a `docs/rules/` page
- [ ] Not applicable

## Checklist

- [ ] Commits follow Conventional Commits and are signed off (`git commit -s`)
- [ ] SPDX header on every new file
