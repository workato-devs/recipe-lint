package lint

import "testing"

func TestCheckRepeatHasWhileCondition(t *testing.T) {
	// repeat with no while_condition child → fires
	malformed := parsedFromJSON(t, `{
		"name": "m", "version": 1, "private": false, "concurrency": 1,
		"code": { "number": 0, "provider": "clock", "name": "scheduled_event", "as": "t",
			"keyword": "trigger", "input": {}, "uuid": "trigger-001", "block": [
			{ "number": 1, "as": "loop", "keyword": "repeat", "uuid": "repeat-001", "block": [
				{ "number": 2, "provider": "salesforce", "name": "s", "as": "a", "keyword": "action", "input": {}, "uuid": "act-001" }
			] }
		] }, "config": [] }`)
	if diags := checkRepeatHasWhileCondition(malformed, "msg"); len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for repeat without while_condition, got %d", len(diags))
	} else if diags[0].Source.JSONPointer != "/code/block/0" {
		t.Errorf("diagnostic pointer = %q, want /code/block/0", diags[0].Source.JSONPointer)
	}

	// repeat with a while_condition child → clean
	correct := parsedFromJSON(t, loopRecipeJSON)
	if diags := checkRepeatHasWhileCondition(correct, "msg"); len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for well-formed repeat, got %d", len(diags))
	}
}

func TestCheckWhileConditionLastInRepeat(t *testing.T) {
	// while_condition followed by another step → fires
	notLast := parsedFromJSON(t, `{
		"name": "m", "version": 1, "private": false, "concurrency": 1,
		"code": { "number": 0, "provider": "clock", "name": "scheduled_event", "as": "t",
			"keyword": "trigger", "input": {}, "uuid": "trigger-001", "block": [
			{ "number": 1, "as": "loop", "keyword": "repeat", "uuid": "repeat-001", "block": [
				{ "number": 2, "keyword": "while_condition", "input": {}, "uuid": "while-001" },
				{ "number": 3, "provider": "salesforce", "name": "s", "as": "a", "keyword": "action", "input": {}, "uuid": "act-001" }
			] }
		] }, "config": [] }`)
	if diags := checkWhileConditionLastInRepeat(notLast, "msg"); len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for while_condition not last, got %d", len(diags))
	}

	// while_condition is last → clean
	correct := parsedFromJSON(t, loopRecipeJSON)
	if diags := checkWhileConditionLastInRepeat(correct, "msg"); len(diags) != 0 {
		t.Errorf("expected 0 diagnostics when while_condition is last, got %d", len(diags))
	}
}

// The if/try structural checks were migrated from the IGM graph to the recipe
// tree layer (Phase 5). This also repaired a latent bug where the IGM-based
// version never fired. Confirm they now flag out-of-order children.
func TestCheckCatchLastInTry_FiresWhenNotLast(t *testing.T) {
	parsed := parsedFromJSON(t, `{
		"name": "m", "version": 1, "private": false, "concurrency": 1,
		"code": { "number": 0, "provider": "workato_recipe_function", "name": "execute", "as": "t",
			"keyword": "trigger", "input": {}, "uuid": "trigger-001", "block": [
			{ "number": 1, "keyword": "try", "uuid": "try-001", "block": [
				{ "number": 2, "provider": "salesforce", "name": "x", "as": "a1", "keyword": "action", "input": {}, "uuid": "act-001" },
				{ "number": 3, "keyword": "catch", "as": "c", "uuid": "catch-001", "block": [] },
				{ "number": 4, "provider": "salesforce", "name": "y", "as": "a2", "keyword": "action", "input": {}, "uuid": "act-002" }
			] }
		] }, "config": [] }`)
	if diags := checkCatchLastInTry(parsed); len(diags) != 1 {
		t.Fatalf("expected 1 CATCH_LAST_IN_TRY diagnostic, got %d", len(diags))
	}
}

func TestCheckElseLastInIf_FiresWhenNotLast(t *testing.T) {
	parsed := parsedFromJSON(t, `{
		"name": "m", "version": 1, "private": false, "concurrency": 1,
		"code": { "number": 0, "provider": "workato_recipe_function", "name": "execute", "as": "t",
			"keyword": "trigger", "input": {}, "uuid": "trigger-001", "block": [
			{ "number": 1, "keyword": "if", "uuid": "if-001", "block": [
				{ "number": 2, "provider": "salesforce", "name": "x", "as": "a1", "keyword": "action", "input": {}, "uuid": "act-001" },
				{ "number": 3, "keyword": "else", "uuid": "else-001", "block": [] },
				{ "number": 4, "provider": "salesforce", "name": "y", "as": "a2", "keyword": "action", "input": {}, "uuid": "act-002" }
			] }
		] }, "config": [] }`)
	if diags := checkElseLastInIf(parsed); len(diags) != 1 {
		t.Fatalf("expected 1 ELSE_LAST_IN_IF diagnostic, got %d", len(diags))
	}
}
