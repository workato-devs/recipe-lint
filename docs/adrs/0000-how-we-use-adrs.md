# How We Use ADRs

**Author(s):** Zayne Turner, Claude [role: assistant; harness: Claude Code; model: Opus 4.8]
**Status:** Accepted
**Date:** June 12, 2026

---

## New to ADRs?

An **Architecture Decision Record** is a short, dated note that captures one significant decision —
what we chose, and *why* — at the moment we made it. You care because it lets you see why the code
is the way it is without reverse-engineering it or interrupting whoever wrote it, and it gives you a
place to argue with the *decision* rather than just the code in front of you. If you've never used
ADRs: read the one that governs the area you're about to touch before you change it, and if your
change proves it wrong, fix the ADR in the same PR (that's the rule in §5).

## Context

This project records architectural decisions as ADRs in `docs/adrs/`. Many are written *ahead of
or alongside* implementation — they capture a **hypothesis** about how something should work, not
settled fact. That's by design; it's how we build in the open. It also creates two risks,
especially as outside and **agent** contributors arrive:

1. A reader — especially an agent — can mistake an `Accepted` ADR for ground truth and build on a
   decision that implementation has since revised. (This nearly happened: ADR 0001 stated "Tier 2
   requires IGM," which turned out to conflate tree-structural checks with control-flow checks.)
2. Without a shared convention, ADRs drift from the code and from each other — inconsistent
   headers, no record of *how* a decision changed, no easy way to see what's still current.

This ADR defines how we treat ADRs so the record stays trustworthy and self-evidently *living*.

## Decision

### 1. ADRs are living hypotheses — verify before relying

An ADR documents the best decision at a point in time, often before the code proves it out.
**Before relying on an ADR's claim, verify it against the current code** — even one marked
`Accepted`. If reality and the ADR disagree, reality wins and the ADR gets amended (below).

### 2. Corrections are in-place, dated amendments

When a decision evolves, **do not rewrite history to look prescient.** Amend the ADR in place:

```markdown
> **Amendment (Month Year): one-line summary of what changed.**
> The original text below said X; implementation showed Y. ...
```

- Keep the original text; layer the amendment on top (a reader should see how understanding moved).
- Add a one-line entry to the ADR's top-of-file **Amendments** log (see header schema).
- If you materially contributed, record yourself per §7 — original authors stay on `Author(s)`;
  a later joiner is added to the `Amended-by` line (never to `Author(s)`).
- ADR 0001 is the worked example — see its "Tier 2: Inter-Step Structure" amendment.

### 3. Status vocabulary

`Status` describes **lifecycle only**, drawn from a fixed set:

| Status | Meaning |
|--------|---------|
| `Proposed` | Decision drafted, not yet agreed. |
| `Accepted` | Agreed and in effect (whether or not fully built). |
| `Superseded` | Replaced by a later ADR. Add `Superseded-by: ADR NNNN`. |

Implementation is a **separate fact**, not a status: record it with an optional `Implemented:`
date line. An ADR can be `Accepted` and not yet implemented, or `Accepted` with an
`Implemented:` date — these are independent.

### 4. Amend the existing ADR, or write a new one?

- **Amend** when the *same* decision evolves, narrows, or is corrected.
- **Write a new ADR** when it's a *different* decision. If the new decision replaces an old one,
  mark the old one `Superseded` (with `Superseded-by:`) and reference it from the new ADR.

Either way, record the **alternatives you considered and why you rejected them** — but mind the
difference between an *architectural no* and a *scope no*. An architectural rejection (we chose Go
over TypeScript; we cut MCP auto-delegation) constrains how the code is built, so it stays inline in
the ADR forever — that's what stops a future contributor from re-litigating it. A scope deferral
("not building X yet") points to the roadmap or a PRD/issue; it isn't an architectural decision and
doesn't belong inline as one.

### 5. Hard rule: contradicting code amends the ADR in the same PR

If a change alters behavior an ADR documents, **amend that ADR in the same pull request.** The PR
template carries a checkbox for this. This is the single rule that keeps the record from rotting.

### 6. Header schema, numbering, template, index

Every ADR starts with:

```markdown
# Title

**Author(s):** Name[, Name, …]     <!-- original decision authors; frozen, does not grow. See §7. -->
**Amended-by:** Name [keys], dir. Name — Month YYYY   <!-- optional; later contributors, one per amendment occasion. See §7. -->
**Status:** Proposed | Accepted | Superseded
**Date:** Month D, YYYY            <!-- original decision date -->
**Implemented:** Month D, YYYY     <!-- optional -->
**Superseded-by:** ADR NNNN        <!-- only if Status: Superseded -->
**References:** ADR NNNN (Title)    <!-- optional -->

**Amendments:**                     <!-- optional; one line per dated amendment -->
- Month YYYY — summary
```

- **Numbering:** `NNNN-kebab-case-title.md`, zero-padded, incremental (this meta-ADR is `0000`).
- **Template:** copy `docs/adrs/TEMPLATE.md` to start a new ADR.
- **Index:** `docs/adrs/README.md` lists every ADR with its status and a one-line summary; update
  it when adding an ADR.

### 7. Authorship & attribution — and *when* an author came on

`Author(s)` and `Amended-by` together make an honest, point-in-time accountability record. Two
layers:

**`Author(s)` — who made the *original* decision.** This line is frozen to the people who authored
the decision at its `Date`. It does **not** grow when someone later amends the ADR.

**`Amended-by` — who *revised* it, and when.** Anyone who joins via a later amendment is recorded
here, never on `Author(s)`. Each entry carries the contributor, their point-in-time provenance, the
directing human, and the amendment date:

```markdown
**Amended-by:** Claude [role: assistant; harness: Claude Code; model: Opus 4.8], dir. Zayne Turner — June 2026
```

Why split them: an agent's `harness`/`model` provenance is only true *at the moment of the
contribution*. Stamping it on a shared `Author(s)` line implies the agent co-authored the original
decision (and that the model applied back then) — both false. Tying it to a dated `Amended-by` entry
keeps the claim honest. Multiple amendments add multiple `Amended-by` lines (one per occasion), each
with its own date.

Two rules carry over for **both** fields:

- **A human is always named.** Every decision and every amendment has at least one accountable
  human. An autonomous agent is never the sole author — the human who deployed or delegated to it
  must appear.
- **Order encodes who did the work.**
  - *Human directing an interactive assistant* (e.g. a person in a Claude Code session): the
    **human leads**, the assistant follows. On `Amended-by`, tag the directing human `dir. Name`.
  - *Autonomous agent acting under delegated judgment*: the **agent leads**, the delegating human
    follows, tagged `(principal)`.

Agent/tool entries carry bracketed keys from a fixed vocabulary — `role`, `harness`, `model` — and
**omit any key you don't know** (don't guess a model). The agent's *identity* (its name) is the
stable key; `harness`/`model` are point-in-time provenance that may change while the identity stays.

```markdown
**Author(s):** Zayne Turner
**Author(s):** Zayne Turner, Chris Miller
**Amended-by:** Claude [role: assistant; harness: Claude Code; model: Opus 4.8], dir. Zayne Turner — June 2026
**Amended-by:** Yoda [role: autonomous agent; harness: Hermes], Jane Doe (principal) — July 2026
```

(The Yoda line omits `model` because it isn't known — the entry is honest about that rather than
guessing.)

## Consequences

- ADRs will visibly carry corrections and amendment logs, with a clear separation between who
  *decided* and who *revised* (and when). That's a **feature**: the record shows how a decision
  actually evolved, so a reader can trust what it says now.
- Contributors (human and agent) get an explicit verify-then-amend protocol, surfaced where they
  work: `AGENTS.md` and the PR template, not buried in a wiki.
- There is a small, deliberate overhead (amend-in-PR, keep the index current). The convention is
  intentionally lightweight — header fields, a template, an index, one PR checkbox — so it is
  followed rather than routed around.
