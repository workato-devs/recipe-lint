package lint

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

// isReturnResponse reports whether a step is an API-platform return_response
// action. Mirrors igm.isExplicitTerminal's return_response branch.
func isReturnResponse(step recipe.FlatStep) bool {
	if step.Code.Provider == nil {
		return false
	}
	return *step.Code.Provider == "workato_api_platform" && step.Code.Name == "return_response"
}

// diffEISFields compares two EIS field lists structurally — by name, type, and
// nested properties — and returns a list of human-readable divergence
// descriptions. labelA/labelB name the two sides (e.g. "input schema" /
// "output schema"). An empty result means the two schemas are identical.
func diffEISFields(a, b []EISField, labelA, labelB, path string) []string {
	var diffs []string

	aByName := make(map[string]EISField, len(a))
	for _, f := range a {
		aByName[f.Name] = f
	}
	bByName := make(map[string]EISField, len(b))
	for _, f := range b {
		bByName[f.Name] = f
	}

	// Field present in A but missing from B (and vice versa). Sort for
	// deterministic output.
	for _, name := range sortedKeys(aByName) {
		if _, ok := bByName[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("field %q is in the %s but missing from the %s", qualify(path, name), labelA, labelB))
		}
	}
	for _, name := range sortedKeys(bByName) {
		if _, ok := aByName[name]; !ok {
			diffs = append(diffs, fmt.Sprintf("field %q is in the %s but missing from the %s", qualify(path, name), labelB, labelA))
		}
	}

	// Fields present in both: compare type, then recurse into properties.
	for _, name := range sortedKeys(aByName) {
		fb, ok := bByName[name]
		if !ok {
			continue
		}
		fa := aByName[name]
		if fa.Type != fb.Type {
			diffs = append(diffs, fmt.Sprintf("field %q type differs: %s has %q, %s has %q",
				qualify(path, name), labelA, orUntyped(fa.Type), labelB, orUntyped(fb.Type)))
		}
		diffs = append(diffs, diffEISFields(fa.Properties, fb.Properties, labelA, labelB, qualify(path, name))...)
	}

	return diffs
}

func sortedKeys(m map[string]EISField) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func qualify(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func orUntyped(t string) string {
	if t == "" {
		return "(untyped)"
	}
	return t
}

// checkReturnResponseSchema validates extended_input_schema (EIS) /
// extended_output_schema (EOS) parity for return_response actions. It backs
// three rule IDs:
//
//   - RETURN_RESPONSE_SCHEMA_PARITY (error): within a single return_response
//     block, EIS and EOS must be structurally identical (names, types, nesting).
//   - RETURN_RESPONSE_SCHEMA_CONSISTENT (warn): across all return_response
//     blocks, every block must share an identical EIS and an identical EOS.
//   - RETURN_RESPONSE_INPUT_MIRROR (warn): EIS must define every top-level key
//     present in the block's input.response.
//
// See issue #17. The Go builtin is required because the divergences are
// structural deep comparisons the declarative assertion vocabulary cannot
// express.
func checkReturnResponseSchema(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic

	type blockSchema struct {
		step recipe.FlatStep
		eis  []EISField
		eos  []EISField
	}
	var blocks []blockSchema

	for _, step := range parsed.Steps {
		if !isReturnResponse(step) {
			continue
		}

		eis, eisErr := parseEIS(step.Code.ExtendedInputSchema)
		eos, eosErr := parseEIS(step.Code.ExtendedOutputSchema)
		if eisErr != nil || eosErr != nil {
			continue // unparseable schema — skip
		}

		// RETURN_RESPONSE_SCHEMA_PARITY: within-block EIS vs EOS.
		if len(eis) > 0 && len(eos) > 0 {
			if divs := diffEISFields(eis, eos, "input schema", "output schema", ""); len(divs) > 0 {
				diags = append(diags, LintDiagnostic{
					Level:   LevelError,
					Message: fmt.Sprintf("return_response extended_input_schema and extended_output_schema diverge: %s", strings.Join(divs, "; ")),
					Source:  &SourceRef{JSONPointer: step.JSONPointer + "/extended_input_schema"},
					RuleID:  "RETURN_RESPONSE_SCHEMA_PARITY",
					Tier:    1,
				})
			}
		}

		// RETURN_RESPONSE_INPUT_MIRROR: every input.response key must be defined
		// under the EIS "response" field's properties. The EIS mirrors the input
		// form, where the response body lives under a top-level "response" field,
		// so input.response keys map to that field's properties — not EIS's
		// top-level names.
		if len(eis) > 0 {
			respSchema := eisFieldProperties(eis, "response")
			for _, key := range responseKeys(step.Code.Input) {
				if !eisTopLevelHas(respSchema, key) {
					diags = append(diags, LintDiagnostic{
						Level:   LevelWarn,
						Message: fmt.Sprintf("input.response field %q is not defined in extended_input_schema", key),
						Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input/response/" + key},
						RuleID:  "RETURN_RESPONSE_INPUT_MIRROR",
						Tier:    1,
					})
				}
			}
		}

		blocks = append(blocks, blockSchema{step: step, eis: eis, eos: eos})
	}

	// RETURN_RESPONSE_SCHEMA_CONSISTENT: every block's EIS (and EOS) must match
	// the first block's. Compare each subsequent block against the first.
	if len(blocks) > 1 {
		first := blocks[0]
		for _, b := range blocks[1:] {
			var divs []string
			divs = append(divs, diffEISFields(first.eis, b.eis, "first block input schema", "this block input schema", "")...)
			divs = append(divs, diffEISFields(first.eos, b.eos, "first block output schema", "this block output schema", "")...)
			if len(divs) > 0 {
				diags = append(diags, LintDiagnostic{
					Level:   LevelWarn,
					Message: fmt.Sprintf("return_response schema differs from the first return_response block (%s); all blocks must share an identical schema: %s", first.step.JSONPointer, strings.Join(divs, "; ")),
					Source:  &SourceRef{JSONPointer: b.step.JSONPointer + "/extended_input_schema"},
					RuleID:  "RETURN_RESPONSE_SCHEMA_CONSISTENT",
					Tier:    1,
				})
			}
		}
	}

	return diags
}

// eisTopLevelHas reports whether any top-level EIS field is named name.
func eisTopLevelHas(fields []EISField, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// eisFieldProperties returns the properties of the top-level EIS field named
// name, or nil if no such field exists.
func eisFieldProperties(fields []EISField, name string) []EISField {
	for _, f := range fields {
		if f.Name == name {
			return f.Properties
		}
	}
	return nil
}

// responseKeys returns the top-level keys of input.response, or nil if input
// has no response object.
func responseKeys(rawInput json.RawMessage) []string {
	if len(rawInput) == 0 {
		return nil
	}
	var inputMap map[string]json.RawMessage
	if err := json.Unmarshal(rawInput, &inputMap); err != nil {
		return nil
	}
	raw, ok := inputMap["response"]
	if !ok {
		return nil
	}
	var respMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &respMap); err != nil {
		return nil // response is not an object (e.g. a datapill string) — skip
	}
	keys := make([]string, 0, len(respMap))
	for k := range respMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
