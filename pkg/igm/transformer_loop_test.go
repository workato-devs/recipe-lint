package igm

import "testing"

func hasEdge(g *Graph, kind EdgeKind, from, to string) bool {
	for _, e := range g.Edges {
		if e.Kind == kind && e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func hasEdgeFrom(g *Graph, from, to string) bool {
	for _, e := range g.Edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

func transformJSON(t *testing.T, src string) *Graph {
	t.Helper()
	g, err := Transform([]byte(src))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	return g
}

// Issue #32: steps around a foreach must stay connected and the foreach's
// alias must resolve.
func TestTransform_Foreach_SequenceAndAlias(t *testing.T) {
	g := transformJSON(t, `{"code": {
		"keyword": "trigger", "provider": "clock", "name": "scheduled_event", "as": "daily", "uuid": "trig",
		"block": [
			{"keyword": "action", "provider": "salesforce", "name": "search_sobjects_soql", "as": "search_rows", "uuid": "search"},
			{"keyword": "foreach", "as": "row", "uuid": "loop", "block": [
				{"keyword": "action", "provider": "logger", "name": "log_message", "as": "log_row", "uuid": "body"}
			]},
			{"keyword": "action", "provider": "logger", "name": "log_message", "as": "log_done", "uuid": "after"}
		]}}`)

	loop := g.NodeByID("loop")
	if loop == nil || loop.Kind != NodeForeach {
		t.Fatalf("foreach node = %+v, want kind foreach", loop)
	}
	if g.AliasMap["row"] != "loop" {
		t.Errorf("alias row = %q, want loop", g.AliasMap["row"])
	}
	if got := g.NodeByID("body"); got == nil || got.ParentID != "loop" {
		t.Errorf("body node parent = %+v, want loop", got)
	}

	for _, want := range []struct {
		kind     EdgeKind
		from, to string
	}{
		{EdgeNext, "search", "loop"},
		{EdgeLoop, "loop", "body"},
		{EdgeLoop, "body", "loop"},
		{EdgeExit, "loop", "after"},
		{EdgeNext, "after", "::end"},
	} {
		if !hasEdge(g, want.kind, want.from, want.to) {
			t.Errorf("missing %s edge %s → %s", want.kind, want.from, want.to)
		}
	}
	if hasEdgeFrom(g, "loop", "::end") || hasEdgeFrom(g, "body", "::end") {
		t.Error("loop and body should not flow to ::end when a step follows the loop")
	}
}

// A loop with no step after it still exits to ::end; its body tail loops back
// instead of dangling.
func TestTransform_Foreach_LastStepExitsToEnd(t *testing.T) {
	g := transformJSON(t, `{"code": {
		"keyword": "trigger", "provider": "clock", "name": "scheduled_event", "uuid": "trig",
		"block": [
			{"keyword": "foreach", "as": "row", "uuid": "loop", "block": [
				{"keyword": "action", "provider": "logger", "name": "log_message", "uuid": "body"}
			]}
		]}}`)

	if !hasEdge(g, EdgeNext, "loop", "::end") {
		t.Error("missing next edge loop → ::end")
	}
	if hasEdgeFrom(g, "body", "::end") {
		t.Error("body tail should loop back, not flow to ::end")
	}
}

// An inner loop that ends the outer loop's body loops back to the outer loop;
// it must not be mistaken for a dangling node.
func TestTransform_Foreach_Nested(t *testing.T) {
	g := transformJSON(t, `{"code": {
		"keyword": "trigger", "provider": "clock", "name": "scheduled_event", "uuid": "trig",
		"block": [
			{"keyword": "foreach", "as": "outer", "uuid": "outer", "block": [
				{"keyword": "foreach", "as": "inner", "uuid": "inner", "block": [
					{"keyword": "action", "provider": "logger", "name": "log_message", "uuid": "body"}
				]}
			]},
			{"keyword": "action", "provider": "logger", "name": "log_message", "uuid": "after"}
		]}}`)

	for _, want := range []struct {
		kind     EdgeKind
		from, to string
	}{
		{EdgeLoop, "outer", "inner"},
		{EdgeLoop, "inner", "body"},
		{EdgeLoop, "body", "inner"},
		{EdgeLoop, "inner", "outer"},
		{EdgeExit, "outer", "after"},
	} {
		if !hasEdge(g, want.kind, want.from, want.to) {
			t.Errorf("missing %s edge %s → %s", want.kind, want.from, want.to)
		}
	}
	if hasEdgeFrom(g, "inner", "::end") {
		t.Error("inner loop should loop back to outer, not flow to ::end")
	}
}

// A terminal step inside a loop body ends the run; it gets no loop-back edge.
func TestTransform_Foreach_TerminalInBody(t *testing.T) {
	g := transformJSON(t, `{"code": {
		"keyword": "trigger", "provider": "workato_api_platform", "name": "receive_request", "uuid": "trig",
		"block": [
			{"keyword": "foreach", "as": "row", "uuid": "loop", "block": [
				{"keyword": "action", "provider": "workato_api_platform", "name": "return_response", "uuid": "ret"}
			]}
		]}}`)

	if hasEdgeFrom(g, "ret", "loop") {
		t.Error("terminal body step should not loop back")
	}
	if !hasEdge(g, EdgeTerminal, "ret", "::end") {
		t.Error("missing terminal edge ret → ::end")
	}
}

func TestTransform_Repeat(t *testing.T) {
	g, err := Transform(loadFixture(t, "repeat_while_correct.recipe.json"))
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	loop := g.NodeByID("page-loop-001")
	if loop == nil || loop.Kind != NodeRepeat {
		t.Fatalf("repeat node = %+v, want kind repeat", loop)
	}
	if g.AliasMap["page_loop"] != "page-loop-001" || g.AliasMap["fetch_page"] != "fetch-page-001" {
		t.Errorf("alias map = %v, want page_loop and fetch_page registered", g.AliasMap)
	}
	if !hasEdge(g, EdgeLoop, "page-loop-001", "fetch-page-001") || !hasEdge(g, EdgeLoop, "fetch-page-001", "page-loop-001") {
		t.Error("missing loop edges between repeat and its body")
	}
	if g.NodeByID("while-cond-001") != nil {
		t.Error("while_condition should not be a graph node")
	}
}
