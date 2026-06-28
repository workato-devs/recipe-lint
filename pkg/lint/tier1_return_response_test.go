package lint

import (
	"encoding/json"
	"testing"

	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

// returnResponseStep builds a workato_api_platform/return_response FlatStep with
// the given EIS, EOS, and input raw JSON (any may be empty).
func returnResponseStep(pointer, eis, eos, input string) recipe.FlatStep {
	c := recipe.Code{
		Keyword:  "action",
		Provider: strPtr("workato_api_platform"),
		Name:     "return_response",
	}
	if eis != "" {
		c.ExtendedInputSchema = json.RawMessage(eis)
	}
	if eos != "" {
		c.ExtendedOutputSchema = json.RawMessage(eos)
	}
	if input != "" {
		c.Input = json.RawMessage(input)
	}
	return recipe.FlatStep{Code: c, JSONPointer: pointer}
}

func TestReturnResponse_SchemaParity_Pass(t *testing.T) {
	schema := `[{"name":"response","type":"object","properties":[{"name":"contact_count","type":"integer"}]}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", schema, schema, ""),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if hasDiag(diags, "RETURN_RESPONSE_SCHEMA_PARITY") {
		t.Error("identical EIS/EOS should not trigger RETURN_RESPONSE_SCHEMA_PARITY")
	}
}

func TestReturnResponse_SchemaParity_TypeMismatch(t *testing.T) {
	eos := `[{"name":"response","type":"object","properties":[{"name":"contact_count","type":"integer"}]}]`
	eis := `[{"name":"response","type":"object","properties":[{"name":"contact_count","type":"string"}]}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", eis, eos, ""),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if !hasDiag(diags, "RETURN_RESPONSE_SCHEMA_PARITY") {
		t.Error("type mismatch (string vs integer) should trigger RETURN_RESPONSE_SCHEMA_PARITY")
	}
}

func TestReturnResponse_SchemaParity_NestedNameMismatch(t *testing.T) {
	eos := `[{"name":"response","type":"object","properties":[{"name":"contacts","type":"array","properties":[{"name":"first_name","type":"string"}]}]}]`
	eis := `[{"name":"response","type":"object","properties":[{"name":"contacts","type":"array","properties":[{"name":"FirstName","type":"string"}]}]}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", eis, eos, ""),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if !hasDiag(diags, "RETURN_RESPONSE_SCHEMA_PARITY") {
		t.Error("nested name divergence (first_name vs FirstName) should trigger RETURN_RESPONSE_SCHEMA_PARITY")
	}
}

func TestReturnResponse_CrossBlock_Consistent(t *testing.T) {
	schema := `[{"name":"response","type":"object","properties":[{"name":"ok","type":"boolean"}]}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", schema, schema, ""),
		returnResponseStep("/code/block/1", schema, schema, ""),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if hasDiag(diags, "RETURN_RESPONSE_SCHEMA_CONSISTENT") {
		t.Error("matching blocks should not trigger RETURN_RESPONSE_SCHEMA_CONSISTENT")
	}
}

func TestReturnResponse_CrossBlock_Divergent(t *testing.T) {
	a := `[{"name":"response","type":"object","properties":[{"name":"ok","type":"boolean"}]}]`
	b := `[{"name":"response","type":"object","properties":[{"name":"ok","type":"string"}]}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", a, a, ""),
		returnResponseStep("/code/block/1", b, b, ""),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if !hasDiag(diags, "RETURN_RESPONSE_SCHEMA_CONSISTENT") {
		t.Error("divergent second block should trigger RETURN_RESPONSE_SCHEMA_CONSISTENT")
	}
}

func TestReturnResponse_InputMirror_Missing(t *testing.T) {
	schema := `[{"name":"response","type":"object","properties":[{"name":"ok","type":"boolean"}]}]`
	// input.response declares an "extra" key that EIS does not define.
	input := `{"response":{"ok":"#{x}","extra":"#{y}"}}`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		returnResponseStep("/code/block/0", schema, schema, input),
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if !hasDiag(diags, "RETURN_RESPONSE_INPUT_MIRROR") {
		t.Error("input.response key absent from EIS should trigger RETURN_RESPONSE_INPUT_MIRROR")
	}
}

func TestReturnResponse_IgnoresNonReturnResponse(t *testing.T) {
	// A salesforce action with mismatched schemas must not be flagged — these
	// rules are scoped to workato_api_platform/return_response only.
	eos := `[{"name":"a","type":"integer"}]`
	eis := `[{"name":"a","type":"string"}]`
	parsed := buildParsedRecipe("test", []recipe.FlatStep{
		{Code: recipe.Code{
			Keyword:              "action",
			Provider:             strPtr("salesforce"),
			Name:                 "create_record",
			ExtendedInputSchema:  json.RawMessage(eis),
			ExtendedOutputSchema: json.RawMessage(eos),
		}, JSONPointer: "/code/block/0"},
	}, nil)
	diags := evalBuiltinRulesForTest(t, parsed)
	if hasDiag(diags, "RETURN_RESPONSE_SCHEMA_PARITY") {
		t.Error("non-return_response action should not trigger RETURN_RESPONSE_SCHEMA_PARITY")
	}
}
