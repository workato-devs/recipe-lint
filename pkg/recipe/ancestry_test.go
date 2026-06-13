package recipe

import "testing"

// nestedRecipe: trigger → [ repeat → [ action, while_condition ], action_after ]
const nestedRecipeJSON = `{
	"name": "nested",
	"version": 1,
	"private": false,
	"concurrency": 1,
	"code": {
		"number": 0, "provider": "clock", "name": "scheduled_event", "as": "trigger1",
		"keyword": "trigger", "input": {}, "uuid": "trigger-001",
		"block": [
			{
				"number": 1, "as": "page_loop", "keyword": "repeat", "uuid": "repeat-001",
				"block": [
					{ "number": 2, "provider": "salesforce", "name": "search", "as": "fetch",
					  "keyword": "action", "input": {}, "uuid": "action-001" },
					{ "number": 3, "keyword": "while_condition", "input": {}, "uuid": "while-001" }
				]
			},
			{ "number": 4, "provider": "logger", "name": "log", "as": "after",
			  "keyword": "action", "input": {}, "uuid": "action-002" }
		]
	},
	"config": []
}`

func mustParse(t *testing.T, data string) *ParsedRecipe {
	t.Helper()
	pr, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return pr
}

func TestAncestry_StepByPointer(t *testing.T) {
	pr := mustParse(t, nestedRecipeJSON)

	if s := pr.StepByPointer("/code"); s == nil || s.Code.Keyword != "trigger" {
		t.Errorf("StepByPointer(/code) = %v, want trigger", s)
	}
	if s := pr.StepByPointer("/code/block/0/block/1"); s == nil || s.Code.Keyword != "while_condition" {
		t.Errorf("StepByPointer(while pointer) = %v, want while_condition", s)
	}
	if s := pr.StepByPointer("/code/block/9"); s != nil {
		t.Errorf("StepByPointer(missing) = %v, want nil", s)
	}
}

func TestAncestry_Parent(t *testing.T) {
	pr := mustParse(t, nestedRecipeJSON)

	// trigger has no parent
	if p := pr.Parent(pr.StepByPointer("/code")); p != nil {
		t.Errorf("Parent(trigger) = %v, want nil", p)
	}
	// while_condition's parent is the repeat
	while := pr.StepByPointer("/code/block/0/block/1")
	if p := pr.Parent(while); p == nil || p.Code.Keyword != "repeat" {
		t.Errorf("Parent(while_condition) = %v, want repeat", p)
	}
	// the trailing action's parent is the trigger
	after := pr.StepByPointer("/code/block/1")
	if p := pr.Parent(after); p == nil || p.Code.Keyword != "trigger" {
		t.Errorf("Parent(after) = %v, want trigger", p)
	}
}

func TestAncestry_Ancestors(t *testing.T) {
	pr := mustParse(t, nestedRecipeJSON)

	action := pr.StepByPointer("/code/block/0/block/0")
	anc := pr.Ancestors(action)
	if len(anc) != 2 {
		t.Fatalf("Ancestors(action) len = %d, want 2", len(anc))
	}
	if anc[0].Code.Keyword != "repeat" {
		t.Errorf("nearest ancestor = %q, want repeat", anc[0].Code.Keyword)
	}
	if anc[1].Code.Keyword != "trigger" {
		t.Errorf("outermost ancestor = %q, want trigger", anc[1].Code.Keyword)
	}
}

func TestAncestry_Children(t *testing.T) {
	pr := mustParse(t, nestedRecipeJSON)

	repeat := pr.StepByPointer("/code/block/0")
	kids := pr.Children(repeat)
	if len(kids) != 2 {
		t.Fatalf("Children(repeat) len = %d, want 2", len(kids))
	}
	// block order preserved; while_condition is last
	if kids[0].Code.Keyword != "action" || kids[1].Code.Keyword != "while_condition" {
		t.Errorf("Children(repeat) = [%q, %q], want [action, while_condition]",
			kids[0].Code.Keyword, kids[1].Code.Keyword)
	}
	// nested action has no children
	if kids := pr.Children(pr.StepByPointer("/code/block/0/block/0")); len(kids) != 0 {
		t.Errorf("Children(leaf action) len = %d, want 0", len(kids))
	}
}
