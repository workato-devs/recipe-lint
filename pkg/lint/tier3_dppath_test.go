package lint

import (
	"encoding/json"
	"testing"

	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

// objectEOS: result{booking_id, status} — a fully-declared object.
const objectEOS = `[{"name":"result","type":"object","properties":[
	{"name":"booking_id","type":"string"},
	{"name":"status","type":"string"}]}]`

// arrayEOS: Contact[] of object{FirstName, Email} — array element fields under properties.
const arrayEOS = `[{"name":"Contact","type":"array","of":"object","properties":[
	{"name":"FirstName","type":"string"},
	{"name":"Email","type":"string"}]}]`

// openObjectEOS: payload is an object with NO declared properties (dynamic/raw JSON).
const openObjectEOS = `[{"name":"payload","type":"object"}]`

func dpPath(line string, path ...interface{}) *DatapillPayload {
	return &DatapillPayload{Line: line, Path: path}
}

func aliasMapWithEOS(alias, eos string) map[string]*recipe.FlatStep {
	step := &recipe.FlatStep{}
	step.Code.As = alias
	if eos != "" {
		step.Code.ExtendedOutputSchema = json.RawMessage(eos)
	}
	return map[string]*recipe.FlatStep{alias: step}
}

func countDPPath(diags []LintDiagnostic) int {
	n := 0
	for _, d := range diags {
		if d.RuleID == "DP_PATH_RESOLVES" {
			n++
		}
	}
	return n
}

func TestCheckDPPathResolves(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		eos     string
		payload *DatapillPayload
		wantHit int
	}{
		{"valid leaf resolves", "step", objectEOS, dpPath("step", "result", "booking_id"), 0},
		{"invented field flagged", "step", objectEOS, dpPath("step", "result", "nope"), 1},
		{"array descent resolves", "search", arrayEOS, dpPath("search", "Contact", "FirstName"), 0},
		{"array index skipped, resolves", "search", arrayEOS, dpPath("search", "Contact", float64(0), "Email"), 0},
		{"array index then invented field flagged", "search", arrayEOS, dpPath("search", "Contact", float64(0), "Nope"), 1},
		{"case mismatch flagged (case-sensitive)", "search", arrayEOS, dpPath("search", "Contact", "firstname"), 1},
		{"open container accepts deeper path", "step", openObjectEOS, dpPath("step", "payload", "anything", "deep"), 0},
		{"absent EOS skipped", "step", "", dpPath("step", "result", "booking_id"), 0},
		{"unresolved alias skipped", "other", objectEOS, dpPath("ghost", "result", "booking_id"), 0},
		{"empty line skipped", "step", objectEOS, dpPath("", "result"), 0},
		{"top-level invented field flagged", "step", objectEOS, dpPath("step", "missing"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliasToStep := aliasMapWithEOS(tt.alias, tt.eos)
			got := countDPPath(checkDPPathResolves(tt.payload, "/code/0/input/x", aliasToStep))
			if got != tt.wantHit {
				t.Errorf("DP_PATH_RESOLVES hits = %d, want %d", got, tt.wantHit)
			}
		})
	}
}

// TestCheckDPPathResolves_RealFixture exercises the rule against a real recipe whose
// Salesforce search step materializes a Contact[...] schema (if_else_branching), confirming
// declared fields resolve and an invented one is flagged.
func TestCheckDPPathResolves_RealFixture(t *testing.T) {
	_, parsed, _ := loadAndBuild(t, "if_else_branching.recipe.json")

	aliasToStep := make(map[string]*recipe.FlatStep, len(parsed.Steps))
	for i := range parsed.Steps {
		if as := parsed.Steps[i].Code.As; as != "" {
			aliasToStep[as] = &parsed.Steps[i]
		}
	}
	if _, ok := aliasToStep["search_contact"]; !ok {
		t.Fatal("expected alias 'search_contact' in fixture")
	}

	// Declared field resolves.
	if n := countDPPath(checkDPPathResolves(dpPath("search_contact", "Contact", "FirstName"), "/p", aliasToStep)); n != 0 {
		t.Errorf("declared field should resolve, got %d hits", n)
	}
	// Invented field is flagged.
	if n := countDPPath(checkDPPathResolves(dpPath("search_contact", "Contact", "EmailAddress"), "/p", aliasToStep)); n != 1 {
		t.Errorf("invented field should be flagged, got %d hits", n)
	}
}
