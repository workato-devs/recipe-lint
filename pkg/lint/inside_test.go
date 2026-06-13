package lint

import (
	"testing"

	"github.com/workato-devs/wk-lint-beta/pkg/recipe"
)

// loopRecipeJSON: trigger → [ repeat → [ logger action, while_condition ], logger action after loop ]
const loopRecipeJSON = `{
	"name": "loop", "version": 1, "private": false, "concurrency": 1,
	"code": {
		"number": 0, "provider": "clock", "name": "scheduled_event", "as": "t",
		"keyword": "trigger", "input": {}, "uuid": "trigger-001",
		"block": [
			{
				"number": 1, "as": "page_loop", "keyword": "repeat", "uuid": "repeat-001",
				"block": [
					{ "number": 2, "provider": "logger", "name": "log", "as": "in_loop",
					  "keyword": "action", "input": {}, "uuid": "log-in-001" },
					{ "number": 3, "keyword": "while_condition", "input": {}, "uuid": "while-001" }
				]
			},
			{ "number": 4, "provider": "logger", "name": "log", "as": "after",
			  "keyword": "action", "input": {}, "uuid": "log-after-001" }
		]
	},
	"config": []
}`

func parsedFromJSON(t *testing.T, data string) *recipe.ParsedRecipe {
	t.Helper()
	pr, err := recipe.Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return pr
}

func TestMatchesWhere_Inside(t *testing.T) {
	pr := parsedFromJSON(t, loopRecipeJSON)

	var inLoop, afterLoop *recipe.FlatStep
	for i := range pr.Steps {
		switch pr.Steps[i].JSONPointer {
		case "/code/block/0/block/0":
			inLoop = &pr.Steps[i]
		case "/code/block/1":
			afterLoop = &pr.Steps[i]
		}
	}
	if inLoop == nil || afterLoop == nil {
		t.Fatal("could not locate test steps")
	}

	sel := &StepSelector{Provider: StringOrArray{"logger"}, Inside: &StepSelector{Keyword: StringOrArray{"repeat"}}}

	if !matchesWhere(pr, inLoop, sel) {
		t.Error("logger inside repeat should match inside:{keyword:repeat}")
	}
	if matchesWhere(pr, afterLoop, sel) {
		t.Error("logger after the loop should NOT match inside:{keyword:repeat}")
	}
}

func TestStepCount_WithInside(t *testing.T) {
	pr := parsedFromJSON(t, loopRecipeJSON)
	zero := 0
	// One logger sits inside the repeat → count is 1, so max:0 fails (rule should fire).
	a := &AssertStepCount{
		Where: &StepSelector{Provider: StringOrArray{"logger"}, Inside: &StepSelector{Keyword: StringOrArray{"repeat"}}},
		Max:   &zero,
	}
	if evalStepCount(pr, a) {
		t.Error("expected step_count max:0 to fail — one logger is inside the loop")
	}
}

func TestValidateStepSelector_RejectsNestedInside(t *testing.T) {
	rule := CustomRule{
		RuleID: "X", Scope: "step", Level: LevelWarn,
		Where: &StepSelector{
			Keyword: StringOrArray{"action"},
			Inside:  &StepSelector{Keyword: StringOrArray{"repeat"}, Inside: &StepSelector{Keyword: StringOrArray{"try"}}},
		},
		Assert: Assertion{FieldExists: &AssertFieldPath{Path: "uuid"}},
	}
	if err := validateCustomRule(rule); err == nil {
		t.Error("expected nested inside-within-inside to be rejected")
	}
}
