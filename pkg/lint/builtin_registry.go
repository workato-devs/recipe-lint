package lint

import (
	"github.com/workato-devs/recipe-lint/pkg/igm"
	"github.com/workato-devs/recipe-lint/pkg/recipe"
)

// BuiltinContext provides everything a builtin rule function might need.
type BuiltinContext struct {
	Parsed    *recipe.ParsedRecipe
	Graph     *igm.Graph
	ConnRules map[string]*ConnectorRules
	Filename  string
	cache     map[string]interface{}
}

// CacheGetOrCompute returns a cached value for key, computing it on first access.
func (c *BuiltinContext) CacheGetOrCompute(key string, compute func() interface{}) interface{} {
	if c.cache == nil {
		c.cache = make(map[string]interface{})
	}
	if v, ok := c.cache[key]; ok {
		return v
	}
	v := compute()
	c.cache[key] = v
	return v
}

// BuiltinFunc is a registered Go function backing a builtin assertion.
// It returns zero or more diagnostics with Message and Source filled in.
// The engine stamps RuleID, Level, and Tier from the JSON rule definition.
//
// This builtin escape hatch is the exception to the "rules are data" principle
// in ADR-0004 (docs/adrs/0004-rules-are-data.md): every rule has a JSON
// definition, and only logic the declarative vocabulary can't yet express
// delegates to a registered Go function. Each builtin added here is a rule that
// requires cutting a new binary — debt against the PRD bar that ADR documents.
// If you change how builtins are registered or dispatched, or grow this set,
// amend ADR-0004 in the same PR.
type BuiltinFunc func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic

var builtinRegistry = map[string]BuiltinFunc{}

func init() {
	RegisterBuiltin("__tier0__", func(ctx *BuiltinContext, rule *CustomRule) []LintDiagnostic {
		return nil
	})
}

// RegisterBuiltin adds a named builtin function to the registry.
func RegisterBuiltin(name string, fn BuiltinFunc) {
	builtinRegistry[name] = fn
}
