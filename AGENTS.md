# AGENTS.md

This file provides guidance to coding agents (and humans) working in this repository. It is the
canonical, tracked instructions file. Vendor-specific files (`CLAUDE.md`, `.cursorrules`, …) are
gitignored and must not be committed.

Claude Code (v2.1.277+) reads this file natively, but by default **only when no `CLAUDE.md` or
`CLAUDE.local.md` exists** in the working directory or above it. Don't create one here. If you
need one locally, start it with `@AGENTS.md`, or this file silently stops loading.

## What This Is

`recipe-lint` is a `wk` CLI plugin that validates Workato recipe JSON files. It ships as a standalone Go binary communicating over JSON-RPC (stdin/stdout) with the `wk` host process. Zero runtime dependencies — single binary, no Node.js.

## Commands

```bash
make build          # Build binary → ./recipe-lint
make test           # Run all tests (go test ./...)
make ci             # What CI runs: build, go vet, gofmt check, tests (uncached). Run before handing off.
make clean          # Remove binary

# Run a single test
go test ./pkg/lint/ -run TestBuiltinRule_ResponseCodesDefined_Pass -v

# Run tests for one package
go test ./pkg/lint/ -v
go test ./pkg/igm/ -v

# Run the linter manually via JSON-RPC (pipe a request to stdin)
echo '{"jsonrpc":"2.0","id":1,"method":"lint.run","params":{"files":["path/to/recipe.json"],"skills_path":"path/to/recipe-skills/skills"}}' | ./recipe-lint

# Install into an isolated wk home to test the real host (never touches ~/.wk)
make build && WK_HOME=$(mktemp -d) sh -c 'wk plugins install . && wk lint path/to/recipe.json'
```

CI (`.github/workflows/ci.yml`) runs `make ci` on pushes and PRs to `develop` and `main`. gofmt drift fails it.

## Architecture

### Pipeline (pkg/lint/lint.go → LintRecipe)

All lint runs flow through `LintRecipe(data, opts)`. The pipeline is:

1. **Load config** — `.wklintrc.json` for severity overrides, ignore patterns
2. **Resolve profile** — embedded profiles (`profiles/*.json`) → plugin-bundled → project-level `.wklint/profiles/`; inheritance via `extends`
3. **Tier 0** — pre-parse schema validation (valid JSON, required top-level keys, config shape, step field presence). Errors here halt the pipeline. Tier 0 rules stay as Go code in `tier0_schema.go` but have catalog-only JSON entries in `builtin_rules.json`.
4. **Parse recipe** — `pkg/recipe.Parse()` produces `ParsedRecipe` with a flat step list and JSON pointers
5. **Load rules** — built-in rules (`builtin_rules.json` embedded via `embed_rules.go`), connector rules from `--skills-path`, custom rules from `.wklint/rules/*.json`. Load problems surface as pipeline diagnostics (see below), never silently.
6. **Tiers 1-3** — all rules flow through `evalCustomRules(ctx, rules, tier)`. Simple rules use composed JSON assertions. Complex rules use `builtin` assertions that delegate to registered Go functions via the builtin registry.
7. **Build IGM graph** — `pkg/igm.Transform()` produces control-flow graph for tier 2-3; if this fails, those tiers are skipped (`IGM_BUILD_FAILED`)
8. **Apply overrides** — profile severity → config severity → filter "off" rules

### Key Packages

- **`cmd/recipe-lint`** — JSON-RPC server and plugin entrypoint. Methods: `lint.run`, `lint.render` (text output for `wk lint`), `lint.pre_push` (hook), `lint.describe_rules` (rule catalog), `lint.version`, `shutdown`.
- **`pkg/lint`** — Core linter. All rule evaluation, profiles, config, custom rule engine.
- **`pkg/recipe`** — Recipe JSON parser. Produces `ParsedRecipe` with flat step list (`FlatStep` with `Code` and `JSONPointer`). Also exposes tree-ancestry lookups (`Parent`/`Ancestors`/`Children`/`StepByPointer`) derived from step JSON pointers.
- **`pkg/igm`** — IGM (Intermediate Graph Model) transformer: recipe JSON → directed graph of nodes and edges for tier 2-3 analysis. Go port of the TypeScript transformer in `recipe-visualizer` (`src/core/transformer.ts`). Models trigger/action/if/try/catch and `foreach`/`repeat` loops (`loop` edges into the body and back, `exit` edge to the next step). A step that yields no node (e.g. `while_condition`) is skipped so its neighbours stay connected. Graph-model changes require an ADR-0001 amendment (see the header of `transformer.go`).
- **`profiles/`** — Embedded JSON profiles (`standard.json`, `strict.json`). Compiled into the binary via `//go:embed`.

### Rule Engine (pkg/lint/custom_rules.go + eval.go + builtin_registry.go)

Every rule — built-in or custom — has a JSON definition. A rule is a JSON object with: `rule_id`, `tier`, `level`, `message`, optional `suggested_fix`, `scope` (recipe|step), optional `where` selector (`keyword`, `provider`, `action_name`, `inside` — one level only), and an `assert` block.

Available assertion types: `field_exists`, `field_absent`, `field_matches`, `field_equals`, `step_count`, `eis_empty`, `eis_field_type`, `all_of`, `any_of`, `not`, `builtin`.

The `builtin` assertion delegates to registered Go functions via `builtin_registry.go`. A `BuiltinFunc` receives a `BuiltinContext` (parsed recipe, IGM graph, connector rules, filename, cache) and returns `[]LintDiagnostic` with source pointers already set. One builtin can back several rule IDs: either compute once with `CacheGetOrCompute` and filter by rule ID (e.g., `check_datapills` backs 7 rule IDs), or dispatch on `rule.RuleID` (e.g., `check_action_name_valid` backs `ACTION_NAME_VALID` and `ACTION_NAME_UNVERIFIED`).

Builtin registrations live in `builtin_tier1.go`, `builtin_tier2.go`, `builtin_tier3.go`. Tier-2 builtins split by concern: control-flow checks use the IGM graph; structural/containment checks (if/try block ordering, repeat/while_condition) use the recipe tree-ancestry layer (`tier2_structure.go`, `tier2_loops.go`).

Field paths use dot notation (`input.response.responses`) resolved by `resolveFieldPath` → `navigateJSON`. Valid roots: `uuid`, `name`, `keyword`, `provider`, `as`, `input`, `extended_input_schema`, `extended_output_schema`, `dynamicPickListSelection`, `filter` (trigger "only continue if…").

Rules are loaded from three sources (in order): embedded `builtin_rules.json`, connector `lint-rules.json` files via `--skills-path`, and project `.wklint/rules/*.json`.

**Pipeline diagnostics** are emitted by the pipeline, not by catalog rules, and have no `builtin_rules.json` entry: `CUSTOM_RULE_INVALID` (unparseable rule file, including a malformed `lint-rules.json`), `SKILLS_PATH_INVALID` (`--skills-path` given but not a directory), `IGM_BUILD_FAILED`. They are documented in `docs/rule-reference.md` and can still be overridden by rule ID.

### Severity Override Chain

Profile rules (lowest) → `.wklintrc.json` rules (highest) → "off" filters out suppressed diagnostics. Profiles support inheritance via `extends`.

## Host Contract (wk)

The host is [`workato-devs/wk`](https://github.com/workato-devs/wk). Verified against wk 1.1.0:

- **`plugin.toml` `version` is display-only.** `wk plugins install`/`list` show it; nothing parses or compares it. It is a static string, so it must be bumped by hand every release (it sat at `0.1.0` through 1.1.0, #30). The binary's version comes separately from GoReleaser's `-X main.version` ldflag.
- **`plugin.toml` `files`** (wk ≥ 1.1.0; older hosts ignore it) lists what install copies besides the manifest and entrypoint. A pattern that matches nothing fails the install, so anything listed must exist in every install layout: the release archive (Homebrew `libexec/`, Scoop app dir) and a source checkout after `make build`. The archive contents are set by `archives.files` in `.goreleaser.yaml`. Keep the two in sync.
- **Text vs JSON output.** `wk lint` calls `lint.render` only for text output. `--json` and the global `--raw` flag bypass it and emit the `lint.run` result. The renderer cannot change the exit code.
- **Reserved flag names.** `raw` is a wk global flag. Don't declare a plugin flag with that name.
- **Pre-push.** Only `error`-level diagnostics fail `lint.pre_push` and block `wk push`. `warn`/`info` never do. Changing a rule's default level to `error` is a user-visible, push-blocking change.

## Design Principles

- **Rules are data (JSON), not code.** Every rule — built-in or custom — should have a JSON definition. Complex rules that require Go implementations use a `builtin` assertion that delegates to registered Go functions. Customers see a uniform rule catalog. See ADR-0004 and the `builtin_rules.json` pattern.
- **Missing connector knowledge is fixed with data, not code.** When a provider lacks validation (e.g., `ACTION_NAME_UNVERIFIED` on `logger`), the fix is a `lint-rules.json` in recipe-skills, not a hardcoded exemption or allowlist in this binary.
- **Prefer false negatives to false positives.** Rules accept conservatively when data is absent, open or dynamic. A rule with false positives gets ignored wholesale, including its true positives (#32). But a skipped check must be visible: silence is not a pass (#31).
- **Zero runtime dependencies.** The binary is self-contained. No Node.js, no external tools. Embedded data files (`formulas.json`, `builtin_rules.json`, `profiles/*.json`) are compiled in via `//go:embed`.
- **Connector rules are data files.** `lint-rules.json` files in skills directories declare per-connector rules. v0.1.0 format has `action_rules`; v0.2.0 format adds `rules[]` array using the custom rule assertion vocabulary. The file's `version` field is overloaded: the Data Tables skill uses `0.3.0` with the old shape. Don't give a version new meaning without resolving that (#24).
- **Dev-only test harnesses must use Go build tags** to keep the zero-dependency install story clean.

## Adding or Changing a Rule

1. Add the JSON entry to `pkg/lint/builtin_rules.json` (or a connector/custom rule file).
2. Add the rule ID to `profiles/standard.json` (`TestBuiltinRules_ProfileCompleteness` fails otherwise) and decide whether `profiles/strict.json` should escalate it (typically `warn` → `error` or `info` → `warn`; rules left out of strict keep their standard level).
3. If it needs Go: prefer routing through an existing builtin over a new `RegisterBuiltin`. Either way, **amend ADR-0004** in the same change. The registered-builtin count and the Go-backed rule-ID count are different metrics, and the amendment must say which one moved.
4. Document it in `docs/rule-reference.md` and add a `CHANGELOG.md` entry under `[Unreleased]`.
5. Test the pass case, the fail case and the "data absent" case. For rules sharing a builtin, add an end-to-end `LintRecipe` test that each rule ID gets only its own diagnostics, with no duplicates.

## Verifying Changes

- `make ci` must pass.
- **Corpus diff for any tier 1-3 behaviour change.** Build the base binary from a temporary worktree (`git worktree add /tmp/rl-base develop && go build -C /tmp/rl-base -o /tmp/rl-old ./cmd/recipe-lint`, then `git worktree remove /tmp/rl-base`), lint a recipe corpus with both binaries (`pkg/lint/testdata/fixtures` plus any real recipes you have, e.g. the templates in a clone of [`workato-devs/recipe-skills`](https://github.com/workato-devs/recipe-skills), passing its `skills/` dir as the skills path), and diff `file / rule_id / pointer`. Every added or removed diagnostic should be explainable. This is how #32's full scope was found: loop-body aliases and `ALL_PATHS_RETURN` were broken too, not just what was reported.
- **Go↔TS IGM parity** (optional, build-tagged; needs a clone of [`workato-devs/recipe-visualizer`](https://github.com/workato-devs/recipe-visualizer) and Node.js): `IGM_TS_PROJECT_DIR=path/to/recipe-visualizer/src go test -tags snapshot ./pkg/igm/ -run Snapshot`. Known mismatch as of 1.2.0: `providerDisplayName` casing for providers without a display-name mapping. `repeat` is Go-only; TS defers it.

## Test Data

- `pkg/lint/testdata/fixtures/` — valid recipe JSON fixtures used by integration tests. **Adding a fixture breaks** `TestLintRunWithDirectory` and `TestLintRunWithMixedFileAndDirectory` in `cmd/recipe-lint/main_test.go`, which hard-code the fixture count. Bump them.
- `pkg/lint/testdata/malformed/` — intentionally broken recipes for tier 0 tests
- `pkg/igm/` — unit tests on graph structure (`transformer_test.go`, `transformer_loop_test.go`), plus the build-tagged Go↔TS snapshot comparison (`snapshot_test.go`; no golden files).
- Fixtures must keep `name` consistent with the filename stem, or `FILENAME_MATCH` fires in end-to-end runs.

## Releases

Contributions (human or agent) go to `develop`, the default branch, via pull request. Maintainers merge `develop` to `main` and cut releases.

1. Bump `version` in `plugin.toml`.
2. Rename `## [Unreleased]` in `CHANGELOG.md` to `## [X.Y.Z] -- YYYY-MM-DD` and start a fresh `[Unreleased]`.
3. Tag `vX.Y.Z`. `release.yml` runs GoReleaser (multi-platform archives with `plugin.toml` and `profiles/*`), then `update-homebrew-and-scoop.yml` updates `workato-devs/homebrew-tap` and `workato-devs/scoop-bucket`.

**Never publish plain `v1.0.7` again.** It was tagged by mistake in June 2026 (the `-beta` suffix was forgotten) and is now marked Pre-release rather than deleted. This is a public Go module, and the Go checksum database is append-only, so re-tagging that version with different content would break checksum verification. If the 1.0.x line is ever revisited, skip from 1.0.6 to 1.0.8. Prerelease tags such as `v1.0.7-beta` are distinct versions.

## Related Repositories

- **`workato-devs/wk`**: the host CLI and plugin protocol. Check its CHANGELOG for manifest and protocol changes on each wk release.
- **`workato-devs/recipe-skills`**: produces the connector `lint-rules.json` files this repo consumes. Contract changes are governed by LABS-0001 in `workato-devs/labs` (`docs/adrs/LABS-0001-skills-lint-rules-as-data.md`), which must be amended alongside them.
- **`workato-devs/recipe-visualizer`**: the TypeScript IGM transformer this repo's `pkg/igm` ports.

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
