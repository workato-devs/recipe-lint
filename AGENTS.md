# AGENTS.md

This file provides guidance to coding agents (and humans) working in this repository. It is the
canonical instructions file; vendor-specific files such as `CLAUDE.md` import it.

## What This Is

`recipe-lint` is a `wk` CLI plugin that validates Workato recipe JSON files. It ships as a standalone Go binary communicating over JSON-RPC (stdin/stdout) with the `wk` host process. Zero runtime dependencies — single binary, no Node.js.

## Commands

```bash
make build          # Build binary → ./recipe-lint
make test           # Run all tests (go test ./...)
make clean          # Remove binary

# Run a single test
go test ./pkg/lint/ -run TestBuiltinRule_ResponseCodesDefined_Pass -v

# Run tests for one package
go test ./pkg/lint/ -v
go test ./pkg/igm/ -v

# Run the linter manually via JSON-RPC (pipe a request to stdin)
echo '{"jsonrpc":"2.0","id":1,"method":"lint.run","params":{"files":["path/to/recipe.json"]}}' | ./recipe-lint
```

## Architecture

### Pipeline (pkg/lint/lint.go → LintRecipe)

All lint runs flow through `LintRecipe(data, opts)`. The pipeline is:

1. **Load config** — `.wklintrc.json` for severity overrides, ignore patterns
2. **Resolve profile** — embedded profiles (`profiles/*.json`) → plugin-bundled → project-level `.wklint/profiles/`; inheritance via `extends`
3. **Tier 0** — pre-parse schema validation (valid JSON, required top-level keys, config shape, step field presence). Errors here halt the pipeline. Tier 0 rules stay as Go code in `tier0_schema.go` but have catalog-only JSON entries in `builtin_rules.json`.
4. **Parse recipe** — `pkg/recipe.Parse()` produces `ParsedRecipe` with a flat step list and JSON pointers
5. **Load rules** — built-in rules (`builtin_rules.json` embedded via `embed_rules.go`), connector rules from `--skills-path`, custom rules from `.wklint/rules/*.json`
6. **Tiers 1-3** — all rules flow through `evalCustomRules(ctx, rules, tier)`. Simple rules use composed JSON assertions. Complex rules use `builtin` assertions that delegate to registered Go functions via the builtin registry.
7. **Build IGM graph** — `pkg/igm.Transform()` produces control-flow graph for tier 2-3; if this fails, those tiers are skipped
8. **Apply overrides** — profile severity → config severity → filter "off" rules

### Key Packages

- **`cmd/recipe-lint`** — JSON-RPC server. Methods: `lint.run`, `lint.pre_push`, `shutdown`. This is the plugin entrypoint.
- **`pkg/lint`** — Core linter. All rule evaluation, profiles, config, custom rule engine.
- **`pkg/recipe`** — Recipe JSON parser. Produces `ParsedRecipe` with flat step list (`FlatStep` with `Code` and `JSONPointer`). Also exposes tree-ancestry lookups (`Parent`/`Ancestors`/`Children`/`StepByPointer`) derived from step JSON pointers.
- **`pkg/igm`** — IGM (Intermediate Graph Model) transformer. Converts recipe JSON → directed graph of nodes and edges for tier 2-3 analysis. Go port of the TypeScript IGM transformer.
- **`profiles/`** — Embedded JSON profiles (`standard.json`, `strict.json`). Compiled into the binary via `//go:embed`.

### Rule Engine (pkg/lint/custom_rules.go + eval.go + builtin_registry.go)

Every rule — built-in or custom — has a JSON definition. A rule is a JSON object with: `rule_id`, `tier`, `level`, `message`, `scope` (recipe|step), optional `where` selector (keyword, provider, action_name, inside), and an `assert` block.

Available assertion types: `field_exists`, `field_absent`, `field_matches`, `field_equals`, `step_count`, `eis_empty`, `eis_field_type`, `all_of`, `any_of`, `not`, `builtin`.

The `builtin` assertion delegates to registered Go functions via `builtin_registry.go`. A `BuiltinFunc` receives a `BuiltinContext` (parsed recipe, IGM graph, connector rules, filename, cache) and returns `[]LintDiagnostic` with source pointers already set. Multi-rule builtins (e.g., `check_datapills` backs 7 rule IDs) use `CacheGetOrCompute` to run analysis once and filter by rule ID.

Builtin registrations live in `builtin_tier1.go`, `builtin_tier2.go`, `builtin_tier3.go`. Tier-2 builtins split by concern: control-flow checks use the IGM graph; structural/containment checks (if/try block ordering, repeat/while_condition) use the recipe tree-ancestry layer (`tier2_structure.go`, `tier2_loops.go`).

Field paths use dot notation (`input.response.responses`) resolved by `resolveFieldPath` → `navigateJSON`.

Rules are loaded from three sources (in order): embedded `builtin_rules.json`, connector `lint-rules.json` files via `--skills-path`, and project `.wklint/rules/*.json`.

### Severity Override Chain

Profile rules (lowest) → `.wklintrc.json` rules (highest) → "off" filters out suppressed diagnostics. Profiles support inheritance via `extends`.

## Design Principles

- **Rules are data (JSON), not code.** Every rule — built-in or custom — should have a JSON definition. Complex rules that require Go implementations use a `builtin` assertion that delegates to registered Go functions. Customers see a uniform rule catalog. See ADR discussions and the `builtin_rules.json` pattern.
- **Zero runtime dependencies.** The binary is self-contained. No Node.js, no external tools. Embedded data files (`formulas.json`, `builtin_rules.json`, `profiles/*.json`) are compiled in via `//go:embed`.
- **Connector rules are data files.** `lint-rules.json` files in skills directories declare per-connector rules. v0.1.0 format has `action_rules`; v0.2.0 format adds `rules[]` array using the custom rule assertion vocabulary.
- **Dev-only test harnesses must use Go build tags** to keep the zero-dependency install story clean.

## Test Data

- `pkg/lint/testdata/fixtures/` — valid recipe JSON fixtures used by integration tests
- `pkg/lint/testdata/malformed/` — intentionally broken recipes for tier 0 tests
- `pkg/igm/` — snapshot tests validate IGM graph output against golden files

## Decision Records (ADRs)

Architectural decisions live in `docs/adrs/` (index: `docs/adrs/README.md`). They are **living
records, not settled truth** — many were written as a hypothesis ahead of implementation.

- **Verify before relying.** Treat an ADR's claims as a hypothesis to check against the current
  code, not as ground truth — even one marked `Accepted`.
- **Amend in the same change.** If your change contradicts what an ADR says, amend that ADR in
  place — a dated `> **Amendment (Month Year): …**` blockquote that preserves the original text —
  as part of the same PR. Don't silently let the record drift.
- **Attribution is point-in-time.** `Author(s)` is frozen to who made the *original* decision; if
  you join by amending, add yourself to `Amended-by` (with your `role`/`harness`/`model` and the
  date), never to `Author(s)`. See §7.

See `docs/adrs/0000-how-we-use-adrs.md` for the full convention (status vocabulary, when to amend
vs. write a new ADR, header schema).
