<!--
ADR template. Copy to docs/adrs/NNNN-kebab-case-title.md (next number, zero-padded) and fill in.
See docs/adrs/0000-how-we-use-adrs.md for the conventions. Delete these comments when done.
-->

# Title

**Author(s):** Name
<!--
Author(s) is a comma-separated list. Always name at least one human. Order encodes who did the work:
  - human + interactive assistant → human first:
    Zayne Turner, Claude [role: assistant; harness: Claude Code; model: Opus 4.8]
  - autonomous agent (delegated judgment) → agent first, accountable human tagged (principal):
    Yoda [role: autonomous agent; harness: Hermes], Jane Doe (principal)
Agent/tool authors use bracketed keys role/harness/model; omit any you don't know. See ADR 0000 §7.
-->

**Status:** Proposed
**Date:** Month D, YYYY
<!-- Optional fields — keep only those that apply:
**Implemented:** Month D, YYYY      (when the decision was actually built)
**Superseded-by:** ADR NNNN         (only if Status: Superseded)
**References:** ADR NNNN (Title)
-->

<!-- Optional; add as the decision evolves (one line per dated amendment, newest first):
**Amendments:**
- Month YYYY — summary of what changed
-->

---

## Context

What problem or need prompted this decision? What constraints are in play? Capture this as the
honest state of knowledge *now* — it's allowed to be a hypothesis that later amendments refine.

## Decision

What we decided, and why this option over the alternatives. Be specific enough that a contributor
(or agent) can act on it and verify it against the code.

## Consequences

What this makes easier, harder, or riskier. Note follow-on work and anything deliberately deferred.

<!--
When this decision later evolves, do NOT rewrite the above. Add a dated amendment in place:

> **Amendment (Month Year): one-line summary.**
> The original text said X; implementation showed Y. ...

Then: (1) add a matching line to the Amendments header field, and (2) if you materially
contributed, append yourself to the Author(s) line per the attribution rules above (ADR 0000 §7) —
an autonomous agent leads with its accountable human tagged (principal).
-->
