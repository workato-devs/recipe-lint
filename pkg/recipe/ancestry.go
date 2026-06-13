package recipe

import "strings"

// The recipe step tree is flat (see Parse), but each step's JSON pointer
// encodes its nesting, e.g. "/code/block/0/block/1". The helpers below recover
// the tree relationships from those pointers so structural rules and custom-rule
// selector containment can ask "what block owns this step?" without a separate
// tree walk.

// pointerParent returns the JSON pointer of a step's block-owner by stripping
// the trailing "/block/N" segment. It returns "" for the root step ("/code"),
// which has no parent.
func pointerParent(pointer string) string {
	idx := strings.LastIndex(pointer, "/block/")
	if idx < 0 {
		return ""
	}
	return pointer[:idx]
}

// StepByPointer returns the step at the given JSON pointer, or nil if none.
func (p *ParsedRecipe) StepByPointer(pointer string) *FlatStep {
	if p == nil || p.stepIndex == nil {
		return nil
	}
	i, ok := p.stepIndex[pointer]
	if !ok {
		return nil
	}
	return &p.Steps[i]
}

// Parent returns the block-owner step that directly contains step, or nil if
// step is the trigger/root or is not part of this recipe.
func (p *ParsedRecipe) Parent(step *FlatStep) *FlatStep {
	if step == nil {
		return nil
	}
	return p.StepByPointer(pointerParent(step.JSONPointer))
}

// Ancestors returns step's block-owner ancestors, nearest first (immediate
// parent, then grandparent, …, up to the trigger). The trigger itself is
// included as the outermost ancestor of any nested step.
func (p *ParsedRecipe) Ancestors(step *FlatStep) []*FlatStep {
	var out []*FlatStep
	for cur := p.Parent(step); cur != nil; cur = p.Parent(cur) {
		out = append(out, cur)
	}
	return out
}

// Children returns the steps directly contained in step's block, in block
// order. Descendants nested deeper than one level are not included.
func (p *ParsedRecipe) Children(step *FlatStep) []*FlatStep {
	if p == nil || step == nil {
		return nil
	}
	prefix := step.JSONPointer + "/block/"
	var out []*FlatStep
	for i := range p.Steps {
		ptr := p.Steps[i].JSONPointer
		if !strings.HasPrefix(ptr, prefix) {
			continue
		}
		// Direct child only: the remainder after "/block/" is a bare index
		// with no further "/block/..." nesting.
		if strings.Contains(ptr[len(prefix):], "/") {
			continue
		}
		out = append(out, &p.Steps[i])
	}
	return out
}
