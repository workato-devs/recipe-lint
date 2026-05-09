package igm

import (
	"os"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../lint/testdata/fixtures/" + name)
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", name, err)
	}
	return data
}

func TestTransform_SimpleConnector(t *testing.T) {
	data := loadFixture(t, "simple_connector.recipe.json")
	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should have nodes: trigger, try, action (update_booking), return_success, catch, log_error, return_error, then branch, ::end
	if len(g.Nodes) == 0 {
		t.Fatal("expected nodes, got none")
	}

	// Check ::end exists
	endNode := g.NodeByID("::end")
	if endNode == nil {
		t.Fatal("missing ::end node")
	}

	// Check trigger exists
	triggerNode := g.NodeByID("trigger-update-booking-001")
	if triggerNode == nil {
		t.Fatal("missing trigger node")
	}
	if triggerNode.Kind != NodeTrigger {
		t.Errorf("trigger kind = %s, want trigger", triggerNode.Kind)
	}

	// Check alias map
	if g.AliasMap["trigger"] != "trigger-update-booking-001" {
		t.Errorf("alias 'trigger' = %q, want trigger-update-booking-001", g.AliasMap["trigger"])
	}
	if g.AliasMap["update_booking"] != "update-booking-action-001" {
		t.Errorf("alias 'update_booking' = %q, want update-booking-action-001", g.AliasMap["update_booking"])
	}
	if g.AliasMap["update_booking_catch"] != "update-booking-catch-001" {
		t.Errorf("alias 'update_booking_catch' = %q, want update-booking-catch-001", g.AliasMap["update_booking_catch"])
	}

	// Check terminal nodes
	terminals := g.TerminalNodes()
	if len(terminals) != 2 {
		t.Errorf("expected 2 terminal nodes, got %d", len(terminals))
	}

	// Check terminal edges to ::end
	termEdges := g.InEdges("::end")
	hasTerminal := false
	for _, e := range termEdges {
		if e.Kind == EdgeTerminal {
			hasTerminal = true
			break
		}
	}
	if !hasTerminal {
		t.Error("expected terminal edges to ::end")
	}

	// All non-end, non-terminal nodes should have outgoing edges
	outgoing := make(map[string]bool)
	for _, e := range g.Edges {
		outgoing[e.From] = true
	}
	for _, n := range g.Nodes {
		if n.ID == "::end" || n.IsTerminal {
			continue
		}
		if !outgoing[n.ID] {
			t.Errorf("node %s (%s) has no outgoing edges", n.ID, n.Kind)
		}
	}

	// Roots should have exactly one entry
	if len(g.Roots) != 1 {
		t.Errorf("expected 1 root, got %d", len(g.Roots))
	}
}

func TestTransform_APIEndpointTryCatch(t *testing.T) {
	data := loadFixture(t, "api_endpoint_try_catch.recipe.json")
	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Check trigger is API endpoint
	triggerNode := g.NodeByID("checkin-trigger-001")
	if triggerNode == nil {
		t.Fatal("missing trigger node")
	}
	if triggerNode.Provider == nil || *triggerNode.Provider != "workato_api_platform" {
		t.Error("trigger should have workato_api_platform provider")
	}

	// Check return_response nodes are terminal
	returnSuccess := g.NodeByID("return-success-checkin-001")
	if returnSuccess == nil {
		t.Fatal("missing return-success-checkin-001")
	}
	if !returnSuccess.IsTerminal {
		t.Error("return_response should be terminal")
	}
	if returnSuccess.HTTPStatus != "200" {
		t.Errorf("HTTPStatus = %q, want 200", returnSuccess.HTTPStatus)
	}

	returnError := g.NodeByID("return-server-error-001")
	if returnError == nil {
		t.Fatal("missing return-server-error-001")
	}
	if !returnError.IsTerminal {
		t.Error("return_response should be terminal")
	}
	if returnError.HTTPStatus != "500" {
		t.Errorf("HTTPStatus = %q, want 500", returnError.HTTPStatus)
	}

	// Check catch node and alias
	if g.AliasMap["checkin_catch"] != "checkin-catch-001" {
		t.Errorf("alias 'checkin_catch' = %q, want checkin-catch-001", g.AliasMap["checkin_catch"])
	}

	// Check if node
	ifNode := g.NodeByID("check-guest-found-001")
	if ifNode == nil {
		t.Fatal("missing if node")
	}
	if ifNode.Kind != NodeIf {
		t.Errorf("if node kind = %s, want if", ifNode.Kind)
	}

	// Check true/false edges from if
	ifEdges := g.OutEdges("check-guest-found-001")
	hasTrue := false
	hasFalse := false
	for _, e := range ifEdges {
		if e.Kind == EdgeTrue {
			hasTrue = true
		}
		if e.Kind == EdgeFalse {
			hasFalse = true
		}
	}
	if !hasTrue {
		t.Error("if node should have true edge")
	}
	if !hasFalse {
		t.Error("if node should have false edge (no else branch)")
	}

	// Check error edge from try to catch
	tryEdges := g.OutEdges("checkin-try-001")
	hasError := false
	for _, e := range tryEdges {
		if e.Kind == EdgeError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("try node should have error edge to catch")
	}
}

func TestTransform_IfElseBranching(t *testing.T) {
	data := loadFixture(t, "if_else_branching.recipe.json")
	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Should have at least one if node
	hasIf := false
	for _, n := range g.Nodes {
		if n.Kind == NodeIf {
			hasIf = true
			break
		}
	}
	if !hasIf {
		t.Error("expected at least one if node")
	}

	// Should have ::end
	if g.NodeByID("::end") == nil {
		t.Fatal("missing ::end node")
	}
}

func TestTransform_DeterministicIDs(t *testing.T) {
	data := loadFixture(t, "simple_connector.recipe.json")

	g1, _ := Transform(data)
	g2, _ := Transform(data)

	if len(g1.Nodes) != len(g2.Nodes) {
		t.Fatalf("node counts differ: %d vs %d", len(g1.Nodes), len(g2.Nodes))
	}
	for i := range g1.Nodes {
		if g1.Nodes[i].ID != g2.Nodes[i].ID {
			t.Errorf("node ID mismatch at %d: %s vs %s", i, g1.Nodes[i].ID, g2.Nodes[i].ID)
		}
	}
	if len(g1.Edges) != len(g2.Edges) {
		t.Fatalf("edge counts differ: %d vs %d", len(g1.Edges), len(g2.Edges))
	}
	for i := range g1.Edges {
		if g1.Edges[i].ID != g2.Edges[i].ID {
			t.Errorf("edge ID mismatch at %d: %s vs %s", i, g1.Edges[i].ID, g2.Edges[i].ID)
		}
	}
}

func TestTransform_GenieTriggerType(t *testing.T) {
	data := []byte(`{
		"name": "Get user manager",
		"code": {
			"keyword": "trigger",
			"provider": "workato_genie",
			"name": "start_workflow",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": {
				"description": "Look up the requesting user's manager",
				"input_schema": "[{\"name\":\"user_email\",\"type\":\"string\"}]",
				"output_schema": "[{\"name\":\"manager_email\",\"type\":\"string\"}]"
			},
			"block": [
				{
					"keyword": "action",
					"provider": "workato_genie",
					"name": "workflow_return_result",
					"as": "return_success",
					"number": 1,
					"uuid": "return-001",
					"input": { "result": { "manager_email": "test@example.com" } }
				}
			]
		},
		"config": [{"keyword": "application", "provider": "workato_genie"}]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	trigger := g.NodeByID("trigger-001")
	if trigger == nil {
		t.Fatal("missing trigger node")
	}
	if trigger.TriggerType != TriggerGenieSkill {
		t.Errorf("TriggerType = %q, want %q", trigger.TriggerType, TriggerGenieSkill)
	}
	if trigger.ProviderDisplayName != "Genie" {
		t.Errorf("ProviderDisplayName = %q, want %q", trigger.ProviderDisplayName, "Genie")
	}
}

func TestTransform_GenieSchemaExtraction(t *testing.T) {
	data := []byte(`{
		"name": "Schema test",
		"code": {
			"keyword": "trigger",
			"provider": "workato_genie",
			"name": "start_workflow",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": {
				"description": "Test skill",
				"input_schema": "[{\"name\":\"email\",\"type\":\"string\"}]",
				"output_schema": "[{\"name\":\"result\",\"type\":\"string\"}]"
			},
			"block": []
		},
		"config": [{"keyword": "application", "provider": "workato_genie"}]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	trigger := g.NodeByID("trigger-001")
	if trigger == nil {
		t.Fatal("missing trigger node")
	}
	if trigger.Kind != NodeTrigger {
		t.Errorf("trigger kind = %s, want trigger", trigger.Kind)
	}
	if trigger.TriggerType != TriggerGenieSkill {
		t.Errorf("TriggerType = %q, want genie_skill", trigger.TriggerType)
	}
}

func TestTransform_WorkflowReturnResultTerminal(t *testing.T) {
	data := []byte(`{
		"name": "Genie return test",
		"code": {
			"keyword": "trigger",
			"provider": "workato_genie",
			"name": "start_workflow",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": { "description": "Test" },
			"block": [
				{
					"keyword": "action",
					"provider": "workato_genie",
					"name": "workflow_return_result",
					"as": "return_success",
					"number": 1,
					"uuid": "return-001",
					"input": { "result": { "success": "true", "manager_email": "a@b.com" } }
				}
			]
		},
		"config": [{"keyword": "application", "provider": "workato_genie"}]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	returnNode := g.NodeByID("return-001")
	if returnNode == nil {
		t.Fatal("missing return node")
	}
	if !returnNode.IsTerminal {
		t.Error("workflow_return_result should be terminal")
	}

	// Should have terminal edge to ::end
	hasTerminalEdge := false
	for _, e := range g.InEdges("::end") {
		if e.From == "return-001" && e.Kind == EdgeTerminal {
			hasTerminalEdge = true
		}
	}
	if !hasTerminalEdge {
		t.Error("expected terminal edge from workflow_return_result to ::end")
	}
}

func TestTransform_StopKeywordTerminal(t *testing.T) {
	data := []byte(`{
		"name": "Stop test",
		"code": {
			"keyword": "trigger",
			"provider": "workato_genie",
			"name": "start_workflow",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": { "description": "Test" },
			"block": [
				{
					"keyword": "stop",
					"number": 1,
					"uuid": "stop-001",
					"input": { "stop_with_error": "false" }
				}
			]
		},
		"config": [{"keyword": "application", "provider": "workato_genie"}]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	stopNode := g.NodeByID("stop-001")
	if stopNode == nil {
		t.Fatal("missing stop node")
	}
	if stopNode.Kind != NodeAction {
		t.Errorf("stop node kind = %s, want action", stopNode.Kind)
	}
	if !stopNode.IsTerminal {
		t.Error("stop keyword should be terminal")
	}

	// Should have terminal edge to ::end
	hasTerminalEdge := false
	for _, e := range g.InEdges("::end") {
		if e.From == "stop-001" && e.Kind == EdgeTerminal {
			hasTerminalEdge = true
		}
	}
	if !hasTerminalEdge {
		t.Error("expected terminal edge from stop to ::end")
	}
}

func TestTransform_ProviderDisplayNames(t *testing.T) {
	data := []byte(`{
		"name": "Display names test",
		"code": {
			"keyword": "trigger",
			"provider": "workato_db_table",
			"name": "new_record_v2",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": {},
			"block": [
				{
					"keyword": "action",
					"provider": "py_eval",
					"name": "invoke_custom_py_code",
					"as": "python_step",
					"number": 1,
					"uuid": "py-001",
					"input": {}
				},
				{
					"keyword": "action",
					"provider": "workato_variable",
					"name": "update_variables",
					"as": "set_var",
					"number": 2,
					"uuid": "var-001",
					"input": {}
				}
			]
		},
		"config": [
			{"keyword": "application", "provider": "workato_db_table"},
			{"keyword": "application", "provider": "py_eval"},
			{"keyword": "application", "provider": "workato_variable"}
		]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	tests := []struct {
		nodeID  string
		display string
	}{
		{"trigger-001", "Data Table"},
		{"py-001", "Python"},
		{"var-001", "Variable"},
	}
	for _, tt := range tests {
		node := g.NodeByID(tt.nodeID)
		if node == nil {
			t.Errorf("missing node %s", tt.nodeID)
			continue
		}
		if node.ProviderDisplayName != tt.display {
			t.Errorf("node %s ProviderDisplayName = %q, want %q", tt.nodeID, node.ProviderDisplayName, tt.display)
		}
	}
}

func TestTransform_StopKeywordInBlock(t *testing.T) {
	data := []byte(`{
		"name": "Stop in if test",
		"code": {
			"keyword": "trigger",
			"provider": "workato_api_platform",
			"name": "api_trigger",
			"as": "trigger",
			"number": 0,
			"uuid": "trigger-001",
			"input": {},
			"block": [
				{
					"keyword": "if",
					"as": "check",
					"number": 1,
					"uuid": "if-001",
					"block": [
						{
							"keyword": "stop",
							"number": 2,
							"uuid": "stop-001",
							"input": { "stop_with_error": "false" }
						},
						{
							"keyword": "else",
							"block": [
								{
									"keyword": "action",
									"provider": "workato_api_platform",
									"name": "return_response",
									"as": "return_ok",
									"number": 3,
									"uuid": "return-001",
									"input": { "http_status_code": "200" }
								}
							]
						}
					]
				}
			]
		},
		"config": [{"keyword": "application", "provider": "workato_api_platform"}]
	}`)

	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	stopNode := g.NodeByID("stop-001")
	if stopNode == nil {
		t.Fatal("missing stop node")
	}
	if !stopNode.IsTerminal {
		t.Error("stop in if branch should be terminal")
	}

	returnNode := g.NodeByID("return-001")
	if returnNode == nil {
		t.Fatal("missing return node")
	}
	if !returnNode.IsTerminal {
		t.Error("return_response should be terminal")
	}

	// Both terminal nodes should have terminal edges to ::end
	terminalEdges := 0
	for _, e := range g.InEdges("::end") {
		if e.Kind == EdgeTerminal {
			terminalEdges++
		}
	}
	if terminalEdges != 2 {
		t.Errorf("expected 2 terminal edges to ::end, got %d", terminalEdges)
	}
}

func TestTransform_InvalidJSON(t *testing.T) {
	_, err := Transform([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTransform_MissingCode(t *testing.T) {
	_, err := Transform([]byte(`{"name": "test"}`))
	if err == nil {
		t.Error("expected error for missing code field")
	}
}

func TestTransform_ParentChildRelationships(t *testing.T) {
	data := loadFixture(t, "simple_connector.recipe.json")
	g, err := Transform(data)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Try node children should include the catch node and action nodes
	tryChildren := g.Children("try-001")
	if len(tryChildren) == 0 {
		t.Error("try node should have children")
	}

	hasCatch := false
	for _, c := range tryChildren {
		if c.Kind == NodeCatch {
			hasCatch = true
		}
	}
	if !hasCatch {
		t.Error("try node should have a catch child")
	}
}
