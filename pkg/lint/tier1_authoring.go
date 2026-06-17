package lint

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

func init() {
	RegisterBuiltin("check_update_vars_raw_form", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		return checkUpdateVarsRawForm(ctx.Parsed)
	})

	RegisterBuiltin("check_body_formula_mode", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		return checkBodyFormulaMode(ctx.Parsed)
	})

	RegisterBuiltin("check_button_params_format", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		return checkButtonParamsFormat(ctx.Parsed)
	})

	RegisterBuiltin("check_stop_missing_reason", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		return checkStopMissingReason(ctx.Parsed)
	})

	RegisterBuiltin("check_genie_authoring", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		allDiags := ctx.CacheGetOrCompute("check_genie_authoring", func() interface{} {
			return checkGenieAuthoring(ctx.Parsed)
		}).([]LintDiagnostic)
		var filtered []LintDiagnostic
		for _, d := range allDiags {
			if d.RuleID == rule.RuleID {
				filtered = append(filtered, d)
			}
		}
		return filtered
	})
}

// checkUpdateVarsRawForm warns when update_variables uses the structured
// variables:[{variable,value}] form instead of input_mode:"raw".
// The structured form is silently dropped by the Workato importer on round-trip.
func checkUpdateVarsRawForm(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic
	for _, step := range parsed.Steps {
		if step.Code.Name != "update_variables" {
			continue
		}
		if len(step.Code.Input) == 0 {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(step.Code.Input, &input); err != nil {
			continue
		}
		if _, hasVariables := input["variables"]; !hasVariables {
			continue
		}
		if modeRaw, hasMode := input["input_mode"]; hasMode {
			var mode string
			if json.Unmarshal(modeRaw, &mode) == nil && mode == "raw" {
				continue
			}
		}
		diags = append(diags, LintDiagnostic{
			Level:   LevelError,
			Message: "update_variables uses structured variables:[{variable,value}] form which is silently dropped on import — use input_mode:\"raw\" instead",
			Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input/variables"},
			RuleID:  "UPDATE_VARS_RAW_FORM",
			Tier:    1,
		})
	}
	return diags
}

// checkBodyFormulaMode warns when an HTTP action body field starts with '='
// (formula mode), which can be silently stripped on import for bodies with
// operator-style keys.
func checkBodyFormulaMode(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic
	for _, step := range parsed.Steps {
		if step.Code.Keyword != "action" {
			continue
		}
		if len(step.Code.Input) == 0 {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(step.Code.Input, &input); err != nil {
			continue
		}
		bodyRaw, ok := input["body"]
		if !ok {
			continue
		}
		var body string
		if json.Unmarshal(bodyRaw, &body) != nil {
			continue
		}
		if strings.HasPrefix(body, "=") {
			diags = append(diags, LintDiagnostic{
				Level:   LevelWarn,
				Message: "Body field uses formula mode (= prefix) which may be silently stripped on import — use #{} interpolation or build the body in a py_eval step",
				Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input/body"},
				RuleID:  "BODY_FORMULA_MODE",
				Tier:    1,
			})
		}
	}
	return diags
}

// checkButtonParamsFormat warns when Workbot button params use URL-encoded
// key=value&key=value format, which Workbot drops silently.
func checkButtonParamsFormat(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic
	for _, step := range parsed.Steps {
		if step.Code.Keyword != "action" {
			continue
		}
		if len(step.Code.Input) == 0 {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(step.Code.Input, &input); err != nil {
			continue
		}
		scanForURLEncodedParams(input, step.JSONPointer+"/input", &diags)
	}
	return diags
}

// scanForURLEncodedParams recursively scans JSON for "params" fields containing
// URL-encoded key=value patterns in button definitions.
func scanForURLEncodedParams(obj map[string]json.RawMessage, basePath string, diags *[]LintDiagnostic) {
	for key, raw := range obj {
		fieldPath := basePath + "/" + key
		if key == "params" {
			var params string
			if json.Unmarshal(raw, &params) == nil && isURLEncodedParams(params) {
				*diags = append(*diags, LintDiagnostic{
					Level:   LevelWarn,
					Message: fmt.Sprintf("Button params uses URL-encoded format %q — use space-separated \"key: value\" format instead", params),
					Source:  &SourceRef{JSONPointer: fieldPath},
					RuleID:  "BUTTON_PARAMS_FORMAT",
					Tier:    1,
				})
			}
			continue
		}
		if key == "attachment_buttons" || key == "buttons" {
			var arr []json.RawMessage
			if json.Unmarshal(raw, &arr) == nil {
				for i, elem := range arr {
					var elemObj map[string]json.RawMessage
					if json.Unmarshal(elem, &elemObj) == nil {
						scanForURLEncodedParams(elemObj, fmt.Sprintf("%s/%d", fieldPath, i), diags)
					}
				}
			}
			continue
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(raw, &nested) == nil {
			scanForURLEncodedParams(nested, fieldPath, diags)
		}
	}
}

// isURLEncodedParams detects the URL-encoded key=value(&key=value)* pattern.
func isURLEncodedParams(s string) bool {
	if s == "" {
		return false
	}
	if !strings.Contains(s, "=") {
		return false
	}
	// URL-encoded: contains '=' but not ': ' (which is the correct format)
	// Heuristic: if it has '&' separators or looks like key=value without colon-space
	if strings.Contains(s, "&") && strings.Contains(s, "=") {
		return true
	}
	// Single key=value without colon — e.g., "action=approve"
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 2 && !strings.Contains(parts[0], ":") && !strings.Contains(parts[0], " ") {
		return true
	}
	return false
}

// checkGenieAuthoring checks genie-skill-specific authoring issues:
// - GENIE_SKILL_DESCRIPTION_EMPTY: empty input.description on start_workflow trigger
// - STOP_ERROR_IN_GENIE: stop with stop_with_error:"true" in a genie recipe
func checkGenieAuthoring(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic

	isGenieRecipe := false
	if len(parsed.Steps) > 0 {
		trigger := parsed.Steps[0]
		if trigger.Code.Keyword == "trigger" && trigger.Code.Provider != nil && *trigger.Code.Provider == "workato_genie" && trigger.Code.Name == "start_workflow" {
			isGenieRecipe = true
			if hasEmptyDescription(trigger.Code.Input) {
				diags = append(diags, LintDiagnostic{
					Level:   LevelWarn,
					Message: "Genie skill trigger has empty input.description — the UI will show a blank skill description and the LLM will not know when to invoke this skill",
					Source:  &SourceRef{JSONPointer: trigger.JSONPointer + "/input/description"},
					RuleID:  "GENIE_SKILL_DESCRIPTION_EMPTY",
					Tier:    1,
				})
			}
		}
	}

	if !isGenieRecipe {
		return diags
	}

	for _, step := range parsed.Steps {
		if step.Code.Keyword != "stop" && !(step.Code.Keyword == "action" && step.Code.Name == "stop") {
			continue
		}
		if len(step.Code.Input) == 0 {
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(step.Code.Input, &input); err != nil {
			continue
		}
		errRaw, ok := input["stop_with_error"]
		if !ok {
			continue
		}
		var stopWithError string
		if json.Unmarshal(errRaw, &stopWithError) != nil {
			continue
		}
		if stopWithError == "true" {
			diags = append(diags, LintDiagnostic{
				Level:   LevelError,
				Message: "stop with stop_with_error:\"true\" is rejected in genie skill recipes — use workflow_return_result with success:false instead",
				Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input/stop_with_error"},
				RuleID:  "STOP_ERROR_IN_GENIE",
				Tier:    1,
			})
		}
	}

	return diags
}

// checkStopMissingReason warns when an error-stop action is missing the
// stop_reason field. Graceful stops (stop_with_error: "false") don't require
// a reason since they represent intentional, non-error termination.
func checkStopMissingReason(parsed *recipe.ParsedRecipe) []LintDiagnostic {
	var diags []LintDiagnostic
	for _, step := range parsed.Steps {
		if step.Code.Keyword != "stop" && !(step.Code.Keyword == "action" && step.Code.Name == "stop") {
			continue
		}
		if len(step.Code.Input) == 0 {
			diags = append(diags, LintDiagnostic{
				Level:   LevelWarn,
				Message: "stop action is missing stop_reason field",
				Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input"},
				RuleID:  "STOP_MISSING_REASON",
				Tier:    1,
			})
			continue
		}
		var input map[string]json.RawMessage
		if err := json.Unmarshal(step.Code.Input, &input); err != nil {
			continue
		}
		if isGracefulStop(input) {
			continue
		}
		if _, ok := input["stop_reason"]; !ok {
			diags = append(diags, LintDiagnostic{
				Level:   LevelWarn,
				Message: "stop action is missing stop_reason field",
				Source:  &SourceRef{JSONPointer: step.JSONPointer + "/input"},
				RuleID:  "STOP_MISSING_REASON",
				Tier:    1,
			})
		}
	}
	return diags
}

func isGracefulStop(input map[string]json.RawMessage) bool {
	raw, ok := input["stop_with_error"]
	if !ok {
		return false
	}
	val := strings.TrimSpace(string(raw))
	return val == `"false"` || val == "false"
}

func hasEmptyDescription(input json.RawMessage) bool {
	if len(input) == 0 {
		return true
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(input, &m); err != nil {
		return true
	}
	descRaw, ok := m["description"]
	if !ok {
		return true
	}
	var desc string
	if json.Unmarshal(descRaw, &desc) != nil {
		return true
	}
	return strings.TrimSpace(desc) == ""
}
