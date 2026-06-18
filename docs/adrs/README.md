# Architecture Decision Records

Architectural decisions for `recipe-lint`. These are **living records** — many were written as a
hypothesis ahead of implementation and carry dated amendments showing how the decision evolved.
Treat an ADR's claims as something to verify against the code, not as settled truth, and amend an
ADR in the same change that contradicts it. See [0000 — How We Use ADRs](0000-how-we-use-adrs.md).

Start a new ADR by copying [`TEMPLATE.md`](TEMPLATE.md) to `NNNN-kebab-case-title.md` (next number).

| ADR | Title | Status | Summary |
|-----|-------|--------|---------|
| [0000](0000-how-we-use-adrs.md) | How We Use ADRs | Accepted | Conventions for ADRs: living hypotheses, in-place dated amendments, status vocabulary, amend-in-PR rule. |
| [0001](0001-tiered-lint-architecture.md) | Tiered Validation for Workato Recipe JSON | Accepted | The four-tier lint pipeline (schema → intra-step → inter-step structure → cross-step data flow) and the Go/plugin architecture. |
| [0002](0002-formula-method-validation.md) | Formula Method Validation | Accepted | Allowlist-based validation of Workato formula methods. |
| [0003](0003-lint-profile-system.md) | Lint Profile System | Accepted | Profiles (`standard`/`strict`) and the severity-override chain with `extends` inheritance. |
| [0004](0004-rules-are-data.md) | Rules Are Data, Not Code | Accepted | Every rule has a JSON definition; declarative assertions need no binary, `builtin` is the escape hatch. Retroactive record of an original-PRD bar — honest about where it's only partially met. |

**Status vocabulary:** `Proposed` (drafted) · `Accepted` (agreed, in effect) · `Superseded`
(replaced by a later ADR). Implementation is tracked separately via an optional `Implemented:`
date in each ADR's header.
