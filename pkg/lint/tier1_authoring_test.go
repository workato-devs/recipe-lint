package lint

import (
	"encoding/json"
	"testing"

	"github.com/workato-devs/wk-lint-beta/pkg/recipe"
)

// --- UPDATE_VARS_RAW_FORM ---

func TestUpdateVarsRawForm_StructuredForm_Fail(t *testing.T) {
	input := map[string]interface{}{
		"variables": []interface{}{
			map[string]interface{}{"variable": "customer_id", "value": "123"},
		},
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "update_variables",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkUpdateVarsRawForm(parsed)
	if !hasDiag(diags, "UPDATE_VARS_RAW_FORM") {
		t.Error("expected UPDATE_VARS_RAW_FORM for structured variables form")
	}
}

func TestUpdateVarsRawForm_RawForm_Pass(t *testing.T) {
	input := map[string]interface{}{
		"input_mode":  "raw",
		"name":        "declare-001:declare_vars:customer_id",
		"customer_id": "123",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "update_variables",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkUpdateVarsRawForm(parsed)
	if hasDiag(diags, "UPDATE_VARS_RAW_FORM") {
		t.Error("expected no UPDATE_VARS_RAW_FORM for raw input form")
	}
}

func TestUpdateVarsRawForm_OtherAction_Pass(t *testing.T) {
	input := map[string]interface{}{
		"variables": []interface{}{
			map[string]interface{}{"variable": "x", "value": "y"},
		},
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "declare_variable",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkUpdateVarsRawForm(parsed)
	if hasDiag(diags, "UPDATE_VARS_RAW_FORM") {
		t.Error("expected no UPDATE_VARS_RAW_FORM for non-update_variables action")
	}
}

// --- BODY_FORMULA_MODE ---

func TestBodyFormulaMode_FormulaPrefix_Fail(t *testing.T) {
	input := map[string]interface{}{
		"body": "={'text' => 'hello'}.to_json",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "make_request_v2",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkBodyFormulaMode(parsed)
	if !hasDiag(diags, "BODY_FORMULA_MODE") {
		t.Error("expected BODY_FORMULA_MODE for = prefix body")
	}
}

func TestBodyFormulaMode_InterpolationBody_Pass(t *testing.T) {
	input := map[string]interface{}{
		"body": "{\"text\":\"#{_dp('{...}')}\"}",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "make_request_v2",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkBodyFormulaMode(parsed)
	if hasDiag(diags, "BODY_FORMULA_MODE") {
		t.Error("expected no BODY_FORMULA_MODE for interpolation body")
	}
}

func TestBodyFormulaMode_Trigger_Pass(t *testing.T) {
	input := map[string]interface{}{
		"body": "={'x' => 1}.to_json",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "trigger",
			Name:    "api_trigger",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code"},
	}, nil)
	diags := checkBodyFormulaMode(parsed)
	if hasDiag(diags, "BODY_FORMULA_MODE") {
		t.Error("expected no BODY_FORMULA_MODE for trigger steps")
	}
}

// --- BUTTON_PARAMS_FORMAT ---

func TestButtonParamsFormat_URLEncoded_Fail(t *testing.T) {
	input := map[string]interface{}{
		"attachment_buttons": []interface{}{
			map[string]interface{}{
				"title":  "Approve",
				"params": "action=approve",
			},
		},
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "post_bot_message",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkButtonParamsFormat(parsed)
	if !hasDiag(diags, "BUTTON_PARAMS_FORMAT") {
		t.Error("expected BUTTON_PARAMS_FORMAT for URL-encoded params")
	}
}

func TestButtonParamsFormat_URLEncodedMulti_Fail(t *testing.T) {
	input := map[string]interface{}{
		"attachment_buttons": []interface{}{
			map[string]interface{}{
				"title":  "Approve",
				"params": "id=123&action=approve",
			},
		},
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "post_bot_message",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkButtonParamsFormat(parsed)
	if !hasDiag(diags, "BUTTON_PARAMS_FORMAT") {
		t.Error("expected BUTTON_PARAMS_FORMAT for multi-param URL-encoded params")
	}
}

func TestButtonParamsFormat_ColonFormat_Pass(t *testing.T) {
	input := map[string]interface{}{
		"attachment_buttons": []interface{}{
			map[string]interface{}{
				"title":  "Approve",
				"params": "action: approve",
			},
		},
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "post_bot_message",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkButtonParamsFormat(parsed)
	if hasDiag(diags, "BUTTON_PARAMS_FORMAT") {
		t.Error("expected no BUTTON_PARAMS_FORMAT for colon-separated params")
	}
}

func TestButtonParamsFormat_NoButtons_Pass(t *testing.T) {
	input := map[string]interface{}{
		"channel": "#general",
		"text":    "Hello",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "post_bot_message",
			Input:   rawJSON(t, input),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkButtonParamsFormat(parsed)
	if hasDiag(diags, "BUTTON_PARAMS_FORMAT") {
		t.Error("expected no BUTTON_PARAMS_FORMAT when no buttons present")
	}
}

// --- GENIE_SKILL_DESCRIPTION_EMPTY ---

func TestGenieSkillDescriptionEmpty_Fail(t *testing.T) {
	genieProvider := "workato_genie"
	input := map[string]interface{}{
		"description":   "",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, input),
		}, JSONPointer: "/code"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if !hasDiag(diags, "GENIE_SKILL_DESCRIPTION_EMPTY") {
		t.Error("expected GENIE_SKILL_DESCRIPTION_EMPTY for empty description")
	}
}

func TestGenieSkillDescriptionEmpty_MissingField_Fail(t *testing.T) {
	genieProvider := "workato_genie"
	input := map[string]interface{}{
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, input),
		}, JSONPointer: "/code"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if !hasDiag(diags, "GENIE_SKILL_DESCRIPTION_EMPTY") {
		t.Error("expected GENIE_SKILL_DESCRIPTION_EMPTY for missing description field")
	}
}

func TestGenieSkillDescription_Populated_Pass(t *testing.T) {
	genieProvider := "workato_genie"
	input := map[string]interface{}{
		"description":   "Look up the requesting user's manager",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, input),
		}, JSONPointer: "/code"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if hasDiag(diags, "GENIE_SKILL_DESCRIPTION_EMPTY") {
		t.Error("expected no GENIE_SKILL_DESCRIPTION_EMPTY for populated description")
	}
}

// --- STOP_ERROR_IN_GENIE ---

func TestStopErrorInGenie_Fail(t *testing.T) {
	genieProvider := "workato_genie"
	triggerInput := map[string]interface{}{
		"description":   "Test skill",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	stopInput := map[string]interface{}{
		"stop_with_error": "true",
		"message":         "Something went wrong",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, triggerInput),
		}, JSONPointer: "/code"},
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   rawJSON(t, stopInput),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if !hasDiag(diags, "STOP_ERROR_IN_GENIE") {
		t.Error("expected STOP_ERROR_IN_GENIE for stop_with_error:true in genie recipe")
	}
}

func TestStopErrorInGenie_GracefulStop_Pass(t *testing.T) {
	genieProvider := "workato_genie"
	triggerInput := map[string]interface{}{
		"description":   "Test skill",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	stopInput := map[string]interface{}{
		"stop_with_error": "false",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, triggerInput),
		}, JSONPointer: "/code"},
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   rawJSON(t, stopInput),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if hasDiag(diags, "STOP_ERROR_IN_GENIE") {
		t.Error("expected no STOP_ERROR_IN_GENIE for graceful stop")
	}
}

func TestStopErrorInGenie_NonGenieRecipe_Pass(t *testing.T) {
	apiProvider := "workato_api_platform"
	triggerInput := map[string]interface{}{
		"response": map[string]interface{}{"responses": []interface{}{}},
	}
	stopInput := map[string]interface{}{
		"stop_with_error": "true",
		"message":         "Error",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &apiProvider,
			Name:     "api_trigger",
			Input:    rawJSON(t, triggerInput),
		}, JSONPointer: "/code"},
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   rawJSON(t, stopInput),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if hasDiag(diags, "STOP_ERROR_IN_GENIE") {
		t.Error("expected no STOP_ERROR_IN_GENIE for non-genie recipe")
	}
}

func TestStopErrorInGenie_StopKeyword_Fail(t *testing.T) {
	genieProvider := "workato_genie"
	triggerInput := map[string]interface{}{
		"description":   "Test skill",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	stopInput := map[string]interface{}{
		"stop_with_error": "true",
		"message":         "Something went wrong",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, triggerInput),
		}, JSONPointer: "/code"},
		{Code: recipe.Code{
			Keyword: "stop",
			Input:   rawJSON(t, stopInput),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if !hasDiag(diags, "STOP_ERROR_IN_GENIE") {
		t.Error("expected STOP_ERROR_IN_GENIE for keyword:stop with stop_with_error:true in genie recipe")
	}
}

// --- STOP_MISSING_REASON ---

func TestStopMissingReason_ErrorStop_NoReason_Fail(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "stop",
			Input:   rawJSON(t, map[string]interface{}{"stop_with_error": "true"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if !hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected STOP_MISSING_REASON for error stop without stop_reason")
	}
}

func TestStopMissingReason_GracefulStop_NoReason_Pass(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "stop",
			Input:   rawJSON(t, map[string]interface{}{"stop_with_error": "false"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("unexpected STOP_MISSING_REASON for graceful stop without stop_reason")
	}
}

func TestStopMissingReason_HasReason_Pass(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "stop",
			Input:   rawJSON(t, map[string]interface{}{"stop_with_error": "false", "stop_reason": "Done processing"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected no STOP_MISSING_REASON when stop_reason is present")
	}
}

func TestStopMissingReason_EmptyInput_Fail(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "stop",
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if !hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected STOP_MISSING_REASON for stop with no input")
	}
}

func TestStopMissingReason_NoStopWithError_Fail(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "stop",
			Input:   rawJSON(t, map[string]interface{}{"message": "something happened"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if !hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected STOP_MISSING_REASON when stop_with_error is absent and stop_reason is missing")
	}
}

func TestStopMissingReason_ActionKeyword_ErrorStop_Fail(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   rawJSON(t, map[string]interface{}{"stop_with_error": "true"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if !hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected STOP_MISSING_REASON for keyword:action name:stop error stop without stop_reason")
	}
}

func TestStopMissingReason_ActionKeyword_GracefulStop_Pass(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   rawJSON(t, map[string]interface{}{"stop_with_error": "false"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("unexpected STOP_MISSING_REASON for graceful stop via action keyword")
	}
}

func TestStopMissingReason_NonStopAction_Pass(t *testing.T) {
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "create_record",
			Input:   rawJSON(t, map[string]interface{}{"table": "accounts"}),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkStopMissingReason(parsed)
	if hasDiag(diags, "STOP_MISSING_REASON") {
		t.Error("expected no STOP_MISSING_REASON for non-stop action")
	}
}

// --- isURLEncodedParams ---

func TestIsURLEncodedParams(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"action=approve", true},
		{"id=123&action=approve", true},
		{"action: approve", false},
		{"approval_id: 1f2c40ee", false},
		{"", false},
		{"just plain text", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isURLEncodedParams(tt.input)
			if got != tt.want {
				t.Errorf("isURLEncodedParams(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Edge case: stop keyword detection ---

func TestStopKeyword_NoInput_NoError(t *testing.T) {
	genieProvider := "workato_genie"
	triggerInput := map[string]interface{}{
		"description":   "Test skill",
		"input_schema":  "[]",
		"output_schema": "[]",
	}
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:  "trigger",
			Provider: &genieProvider,
			Name:     "start_workflow",
			Input:    rawJSON(t, triggerInput),
		}, JSONPointer: "/code"},
		{Code: recipe.Code{
			Keyword: "action",
			Name:    "stop",
			Input:   json.RawMessage(`{}`),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := checkGenieAuthoring(parsed)
	if hasDiag(diags, "STOP_ERROR_IN_GENIE") {
		t.Error("expected no STOP_ERROR_IN_GENIE for stop without stop_with_error field")
	}
}
