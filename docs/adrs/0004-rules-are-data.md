# Rules Are Data, Not Code

**Author(s):** Zayne Turner, Claude [role: assistant; harness: Claude Code; model: Opus 4.8]
**Amended-by:** Claude [role: assistant; harness: Claude Code; model: Opus 4.8], dir. Zayne Turner — June 2026

**Status:** Accepted
**Date:** June 17, 2026
**Implemented:** Partially — see Consequences
**References:** ADR 0001 (Tiered Validation Architecture); Labs repo `labs/docs/adrs/LABS-0001-skills-lint-rules-as-data.md` (skills-side rules contract)

**Amendments:**
- June 2026 — Recorded the `lint-rules.json` contract's downstream consumer (the agent-skills repo) and pointed to the Labs-repo ADR that owns that cross-repo decision.

---

## Context

This decision originated in the **original product requirements**, well before ADR-0001 was
written, but it was never captured as its own decision record. That omission is the reason this ADR
exists now, and it is the honest subject of the record below.

The PRD set a specific, load-bearing bar:

> **Changing the rules a customer runs must not require cutting a new Go binary.**

The intent was operational: connector teams and customers should be able to add, tune, and ship
lint rules as *data* — on their own cadence — without a release of `recipe-lint` itself. A rule was
meant to be a JSON artifact, not a code change.

Because this bar was never written down as an ADR, it was not visible as a hard constraint during
scoping. The early scoping work (done by an agent) interpreted the absence of a stated requirement
as license to treat "rules are data" as an aspiration rather than a contract, and leaned on a "no
DSL" argument to justify implementing non-trivial rules directly in Go. That interpretation went
unchallenged precisely because there was no decision record to check it against — the exact failure
mode the ADR convention exists to prevent (ADR-0000: *verify before relying*).

## Decision

**Every rule — built-in or custom — has a JSON definition.** Customers and connector teams see a
single uniform rule catalog and author rules as data.

Two mechanisms back this:

1. **A declarative assertion vocabulary**, interpreted at runtime in `pkg/lint/eval.go`:
   `field_exists`, `field_absent`, `field_matches`, `field_equals`, `step_count`, `eis_empty`,
   `eis_field_type`, `all_of`, `any_of`, `not`. Rules expressible in this vocabulary require **no
   new binary** — they load from embedded `builtin_rules.json`, from skills `lint-rules.json`
   (`--skills-path`), or from project `.wklint/rules/*.json`.

2. **A `builtin` assertion** that delegates to a Go function registered in the binary
   (`pkg/lint/builtin_registry.go`). This is the escape hatch for logic the declarative vocabulary
   cannot yet express (control-flow analysis over the IGM graph, datapill tracing, etc.).

The principle is that the declarative path is the default and the `builtin` path is the exception.

## Alternatives considered

- **A full rule DSL.** Rejected ("no DSL"). A bespoke expression language is a large surface to
  design, document, version, and secure, and most rules don't need it. *This rejection is sound on
  its own merits — the failure was not the "no DSL" conclusion, but using it as cover to skip the
  declarative-data layer entirely and route complex rules straight into Go.*
- **All rules as Go code.** Rejected: directly violates the PRD bar; every connector rule change
  would force a `recipe-lint` release.

## Consequences

**Honest status of the PRD bar: partially met, and that gap traces directly to this ADR's absence.**

- Rules expressible in the declarative vocabulary meet the bar fully: they ship as data, no binary.
- Any rule needing a `builtin` assertion does **not** meet the bar — it requires a Go function
  compiled into the binary. As of this writing there are **27 registered builtins**
  (`grep -rh 'RegisterBuiltin("' pkg/lint/*.go | grep -v '__tier0__'`, excluding the no-op
  placeholder), and they back the bulk of the non-trivial checks. In practice, "add a real rule"
  still often means "cut a new binary," which is the outcome the PRD set out to avoid.
- The drift was gradual and unchallenged because there was no record stating the bar. Had this ADR
  existed, the volume of `builtin`-backed rules would have been visible as debt against an explicit
  constraint, not an unexamined default.

**Follow-on work (deliberately surfaced, not yet decided):**

- Decide whether closing the gap means *expanding the declarative vocabulary* (more assertion types
  so fewer rules need Go) or *accepting a bounded set of builtins as legitimate* and documenting
  which rule classes are permitted to be code. Either is a real decision and should get its own ADR
  or an amendment here — not another silent default.
- Track the builtin count as a debt metric against the PRD bar.

> **Amendment (June 2026): the `lint-rules.json` contract has a downstream consumer — the agent-skills repo.**
> The original record above treats "skills `lint-rules.json` (`--skills-path`)" only as one of the
> *sources* a rule can load from. That understates the relationship. The agent-skills repo declares,
> per skill, which lint rules apply to it (e.g. `skills/workato-recipes/variable/lint-rules.json`) —
> a decision that repo took on given this linter's existence, to make its pre-commit checks easier.
> That makes the skills repo the **living proof of this ADR's bar**: rules shipping as data, on a
> separate repo's own cadence, with no `recipe-lint` binary cut.
>
> The consequence for anyone amending this ADR: the declarative assertion vocabulary and the
> `lint-rules.json` shape are a **contract with a standing downstream consumer.** Changing or
> removing assertion types, or altering how `--skills-path` files are read, has a blast radius
> beyond this repo. The cross-repo decision itself — what the skills repo declares and why — is owned
> and recorded in the Labs repo (`labs/docs/adrs/LABS-0001-skills-lint-rules-as-data.md`), not here;
> this ADR governs only the linter-side schema it consumes.

<!-- When this evolves, add a dated amendment in place; do not rewrite the above. -->
