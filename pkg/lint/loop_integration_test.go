package lint

import (
	"fmt"
	"os"
	"testing"

	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

func ruleIDs(diags []LintDiagnostic) map[string]bool {
	m := make(map[string]bool)
	for _, d := range diags {
		m[d.RuleID] = true
	}
	return m
}

// End-to-end coverage of issue #13: the malformed repro must surface the loop
// structural rules, while the well-formed repro must be free of them.
func TestLoopRules_EndToEnd(t *testing.T) {
	malformed, err := os.ReadFile("testdata/malformed/repeat_while_malformed.recipe.json")
	if err != nil {
		t.Fatal(err)
	}
	diags, err := LintRecipe(malformed, LintOptions{Tiers: []int{1, 2}})
	if err != nil {
		t.Fatalf("LintRecipe(malformed): %v", err)
	}
	got := ruleIDs(diags)
	for _, want := range []string{"REPEAT_NO_PROVIDER", "REPEAT_HAS_WHILE_CONDITION"} {
		if !got[want] {
			t.Errorf("malformed repro: expected %s to fire", want)
		}
	}

	correct, err := os.ReadFile("testdata/fixtures/repeat_while_correct.recipe.json")
	if err != nil {
		t.Fatal(err)
	}
	diags, err = LintRecipe(correct, LintOptions{Tiers: []int{1, 2}})
	if err != nil {
		t.Fatalf("LintRecipe(correct): %v", err)
	}
	got = ruleIDs(diags)
	for _, unwanted := range []string{"REPEAT_NO_PROVIDER", "WHILE_CONDITION_NO_PROVIDER", "REPEAT_HAS_WHILE_CONDITION", "WHILE_CONDITION_LAST_IN_REPEAT"} {
		if got[unwanted] {
			t.Errorf("well-formed repro: %s should not fire", unwanted)
		}
	}
}

// Non-loop containment (#14 acceptance sketch): the if_else_branching fixture
// has logger/return actions nested inside a catch. inside:{keyword:"catch"}
// should match them and not the actions outside the catch.
func TestInside_Catch(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/if_else_branching.recipe.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := recipe.Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	sel := &StepSelector{Keyword: StringOrArray{"action"}, Inside: &StepSelector{Keyword: StringOrArray{"catch"}}}
	var inCatch, outsideCatch int
	for i := range parsed.Steps {
		step := &parsed.Steps[i]
		if step.Code.Keyword != "action" {
			continue
		}
		if matchesWhere(parsed, step, sel) {
			inCatch++
		} else {
			outsideCatch++
		}
	}
	if inCatch != 2 {
		t.Errorf("expected 2 actions inside catch, got %d", inCatch)
	}
	if outsideCatch == 0 {
		t.Errorf("expected some actions outside catch, got %d", outsideCatch)
	}
}

// Issue #32: the foreach alias resolves, and steps on either side of the loop
// stay reachable. The repo's own repeat fixture had the same false positive.
func TestLoopDataFlow_NoFalsePositives(t *testing.T) {
	for _, name := range []string{"foreach_dataflow.recipe.json", "repeat_while_correct.recipe.json"} {
		data, err := os.ReadFile("testdata/fixtures/" + name)
		if err != nil {
			t.Fatal(err)
		}
		diags, err := LintRecipe(data, LintOptions{Tiers: []int{2, 3}})
		if err != nil {
			t.Fatalf("LintRecipe(%s): %v", name, err)
		}
		for _, d := range diags {
			switch d.RuleID {
			case "DP_LINE_RESOLVES", "DP_STEP_REACHABLE", "DP_PROVIDER_MATCHES":
				t.Errorf("%s: unexpected %s at %s: %s", name, d.RuleID, d.Source.JSONPointer, d.Message)
			}
		}
	}
}

// Loops are real control flow for ALL_PATHS_RETURN: a return after the loop
// covers the path; a loop that ends the recipe is the dangling path.
func TestLoopAllPathsReturn(t *testing.T) {
	const tmpl = `{"name": "loop return", "version": 1, "private": true, "concurrency": 1,
		"config": [{"keyword": "application", "provider": "workato_api_platform", "skip_validation": false, "account_id": null}],
		"code": {"number": 0, "keyword": "trigger", "provider": "workato_api_platform", "name": "receive_request", "as": "trigger", "uuid": "trig", "input": {},
		"block": [
			{"number": 1, "keyword": "action", "provider": "logger", "name": "log_message", "as": "before", "uuid": "before", "input": {}},
			{"number": 2, "keyword": "foreach", "as": "row", "uuid": "loop", "input": {}, "block": [
				{"number": 3, "keyword": "action", "provider": "logger", "name": "log_message", "as": "body", "uuid": "body", "input": {}}
			]}%s
		]}}`
	const ret = `,
			{"number": 4, "keyword": "action", "provider": "workato_api_platform", "name": "return_response", "as": "ret", "uuid": "ret", "input": {"http_status_code": "200"}}`

	allPathsReturn := func(src string) []string {
		t.Helper()
		diags, err := LintRecipe([]byte(src), LintOptions{Tiers: []int{2}})
		if err != nil {
			t.Fatalf("LintRecipe: %v", err)
		}
		var ptrs []string
		for _, d := range diags {
			if d.RuleID == "ALL_PATHS_RETURN" {
				ptrs = append(ptrs, d.Source.JSONPointer)
			}
		}
		return ptrs
	}

	if got := allPathsReturn(fmt.Sprintf(tmpl, ret)); len(got) != 0 {
		t.Errorf("return after loop: unexpected ALL_PATHS_RETURN at %v", got)
	}
	if got := allPathsReturn(fmt.Sprintf(tmpl, "")); len(got) != 1 || got[0] != "/code/block/1" {
		t.Errorf("loop ends recipe: ALL_PATHS_RETURN at %v, want exactly [/code/block/1]", got)
	}
}
