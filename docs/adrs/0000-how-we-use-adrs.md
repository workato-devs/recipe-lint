# How We Use ADRs

**Author(s):** Zayne Turner, Claude [role: assistant; harness: Claude Code; model: Opus 4.8]
**Status:** Accepted
**Date:** June 12, 2026

---

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
- If you materially contributed, **append yourself to the `Author(s)` line** (see §7 for how to
  attribute humans vs. agents) — it's a list, so ADRs accrue authors as they evolve.
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

### 5. Hard rule: contradicting code amends the ADR in the same PR

If a change alters behavior an ADR documents, **amend that ADR in the same pull request.** The PR
template carries a checkbox for this. This is the single rule that keeps the record from rotting.

### 6. Header schema, numbering, template, index

Every ADR starts with:

```markdown
# Title

**Author(s):** Name[, Name, …]     <!-- comma-separated; grows as people/agents amend. See §7. -->
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

### 7. Authorship & attribution

`Author(s)` is a comma-separated list. Two rules make it an honest accountability record as agents
start contributing:

- **A human is always named.** Every ADR has at least one human author. An autonomous agent is
  never the sole author — the human who deployed or delegated to it is accountable and must appear.
- **Order encodes who did the work.** List the entity that exercised the judgment first:
  - *Human directing an interactive assistant* (e.g. a person in a Claude Code session): the
    **human leads**, the assistant follows. The human did the work; the tool helped.
  - *Autonomous agent acting under delegated judgment* (e.g. a supervisor agent): the **agent
    leads**, the delegating human follows, tagged `(principal)`. The agent did the work; this is a
    blame/audit trail, so the actor of record is the agent and the principal is the accountable
    human behind it.

Agent and tool authors carry bracketed keys from a fixed vocabulary — `role`, `harness`, `model` —
and **omit any key you don't know** (don't guess a model). The agent's *identity* (its name) is the
stable key; `harness`/`model` are point-in-time provenance that may change while the identity stays.

```markdown
**Author(s):** Zayne Turner
**Author(s):** Zayne Turner, Claude [role: assistant; harness: Claude Code; model: Opus 4.8]
**Author(s):** Yoda [role: autonomous agent; harness: Hermes], Jane Doe (principal)
**Author(s):** Yoda [role: autonomous agent; harness: Hermes; model: Opus 4.8], Jane Doe (principal)
```

(Line 3 omits `model` because it isn't known — the entry is honest about that rather than guessing.)

## Consequences

- ADRs will visibly carry corrections and amendment logs. That's a **feature**: the record shows
  how a decision actually evolved, so a reader can trust what it says now.
- Contributors (human and agent) get an explicit verify-then-amend protocol, surfaced where they
  work: `AGENTS.md` and the PR template, not buried in a wiki.
- There is a small, deliberate overhead (amend-in-PR, keep the index current). The convention is
  intentionally lightweight — header fields, a template, an index, one PR checkbox — so it is
  followed rather than routed around.
