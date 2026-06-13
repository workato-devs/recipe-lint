package lint

import (
	"os"
	"testing"

	"github.com/workato-devs/wk-lint-beta/pkg/recipe"
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
