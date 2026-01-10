package checks

import (
	"GoParser/model"
	"go/ast"
	"go/types"
)

type AnalysisContext struct {
	TypeBoundsInfo    map[string]TypeBoundInfo
	LocalGenerics     map[string]*GenericDefinition
	LocalGenericTypes map[string]bool
	TypeInfo          *types.Info
	File              *ast.File
}

type TypeBoundInfo struct {
	HasNonTrivialBound bool
	HasStructBound     bool
}

type GenericDefinition struct {
	Name          string
	IsMethod      bool
	NumTypeParams int
}

type ASTCheck interface {
	Check(node ast.Node, context *AnalysisContext) bool
	Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext)
}

type CheckRunner struct {
	checks []ASTCheck
}

func NewCheckRunner(checks []ASTCheck) *CheckRunner {
	return &CheckRunner{checks: checks}
}

func (r *CheckRunner) RunChecks(file *ast.File, counters *model.GenericCounters, context *AnalysisContext) {
	context.File = file
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		for _, check := range r.checks {
			if check.Check(n, context) {
				check.Update(counters, n, context)
			}
		}
		return true
	})
}