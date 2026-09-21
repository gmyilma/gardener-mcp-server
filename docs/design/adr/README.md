# Architecture decision records

Short records of decisions that are expensive to reverse, written down so that
later contributors can see *why* rather than guessing from the code.

An ADR is immutable once accepted. If a decision changes, write a new ADR and
mark the old one superseded. Editing history to look consistent destroys the
reason the record exists.

| ADR | Title | Status |
|---|---|---|
| [0001](0001-deployment-and-identity-modes.md) | Deployment and identity modes | Accepted |
| [0002](0002-mcp-sdk-selection.md) | Use the official MCP Go SDK | Accepted |
| [0003](0003-gardener-dependency-strategy.md) | Depend on Gardener's API module, not its root module | Accepted |
| [0004](0004-evidence-envelope-and-trust-labelling.md) | Evidence envelope and trust labelling | Accepted |
| [0005](0005-no-resource-mutation-path.md) | No resource-mutation path | Accepted |

The supporting analysis for all five is in
[../architecture-review.md](../architecture-review.md).

## Writing a new one

Copy [`template.md`](template.md), take the next number, add a row above. Keep
it to one page. The Consequences section is the valuable part — anyone can
record what was decided; the useful record says what it costs.
