package lint

import "github.com/workato-devs/recipe-lint/pkg/recipe"

// Loop structural checks. Unlike the if/try checks in tier2_structure.go, these
// run on the recipe tree-ancestry layer (parsed.Children) rather than the IGM
// graph — loop containment is a syntactic question, not a control-flow one, and
// the IGM does not model repeat/while_condition.

// checkRepeatHasWhileCondition verifies that every "repeat" block contains a
// "while_condition" child — its absence is the malformation that silently fails
// UI reconstruction after push.
// Rule: REPEAT_HAS_WHILE_CONDITION
func checkRepeatHasWhileCondition(parsed *recipe.ParsedRecipe, message string) []LintDiagnostic {
	var diags []LintDiagnostic
	for i := range parsed.Steps {
		step := &parsed.Steps[i]
		if step.Code.Keyword != "repeat" {
			continue
		}
		hasWhile := false
		for _, child := range parsed.Children(step) {
			if child.Code.Keyword == "while_condition" {
				hasWhile = true
				break
			}
		}
		if !hasWhile {
			diags = append(diags, LintDiagnostic{
				Message: message,
				Source:  &SourceRef{JSONPointer: step.JSONPointer},
			})
		}
	}
	return diags
}

// checkWhileConditionLastInRepeat verifies that, within a "repeat" block, the
// "while_condition" exit child is the last child.
// Rule: WHILE_CONDITION_LAST_IN_REPEAT
func checkWhileConditionLastInRepeat(parsed *recipe.ParsedRecipe, message string) []LintDiagnostic {
	var diags []LintDiagnostic
	for i := range parsed.Steps {
		step := &parsed.Steps[i]
		if step.Code.Keyword != "repeat" {
			continue
		}
		children := parsed.Children(step)
		for j := range children {
			if children[j].Code.Keyword != "while_condition" {
				continue
			}
			if j != len(children)-1 {
				diags = append(diags, LintDiagnostic{
					Message: message,
					Source:  &SourceRef{JSONPointer: children[j].JSONPointer},
				})
			}
		}
	}
	return diags
}
