package checks

import (
	"GoParser/model"
	"go/ast"
	"go/types"
	"strings"
)

// GenericFuncCallCheck counts generic function call sites (CallExpr).
// It increments GenericFuncInstantiationExplicit or GenericFuncInstantiationInferred.
type GenericFuncCallCheck struct {
	handlers []ExpressionHandler
}

func NewGenericFuncCallCheck() *GenericFuncCallCheck {
	return &GenericFuncCallCheck{
		handlers: []ExpressionHandler{
			&IndexExprHandler{},
			&IndexListExprHandler{},
			&InferredIdentHandler{},
			&InferredSelectorHandler{},
		},
	}
}

func (f *GenericFuncCallCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.CallExpr)
	return ok && context.TypeInfo != nil
}

func (f *GenericFuncCallCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	callExpr := node.(*ast.CallExpr)
	instContext := &InstantiationContext{
		TypeInfo:          context.TypeInfo,
		LocalGenerics:     context.LocalGenerics,
		LocalGenericTypes: context.LocalGenericTypes,
	}

	for _, handler := range f.handlers {
		if handler.CanHandle(callExpr.Fun) {
			isExternal := handler.IsExternal(callExpr.Fun, instContext)
			if isExternal {
				return
			}

			isExplicit := handler.IsExplicit()

			if !isExplicit {
				if !f.hasInstance(callExpr.Fun, instContext) {
					return
				}
			}

			if isExplicit {
				counters.GenericFuncInstantiationExplicit++
			} else {
				counters.GenericFuncInstantiationInferred++
			}
			return
		}
	}
}

func (f *GenericFuncCallCheck) hasInstance(expr ast.Expr, context *InstantiationContext) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		_, hasInstance := context.TypeInfo.Instances[ident]
		return hasInstance
	}
	if selExpr, ok := expr.(*ast.SelectorExpr); ok {
		_, hasInstance := context.TypeInfo.Instances[selExpr.Sel]
		return hasInstance
	}
	return false
}

// GenericTypeCompositeLitCheck counts generic type instantiations via composite literals (CompositeLit).
// It increments GenericTypeInstantiationExplicit or GenericTypeInstantiationInferred.
type GenericTypeCompositeLitCheck struct {
	handlers []ExpressionHandler
}

func NewGenericTypeCompositeLitCheck() *GenericTypeCompositeLitCheck {
	return &GenericTypeCompositeLitCheck{
		handlers: []ExpressionHandler{
			&IndexExprHandler{},
			&IndexListExprHandler{},
			&InferredIdentHandler{},
		},
	}
}

func (t *GenericTypeCompositeLitCheck) Check(node ast.Node, context *AnalysisContext) bool {
	compLit, ok := node.(*ast.CompositeLit)
	return ok && compLit.Type != nil && context.TypeInfo != nil
}

func (t *GenericTypeCompositeLitCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	compLit := node.(*ast.CompositeLit)
	instContext := &InstantiationContext{
		TypeInfo:          context.TypeInfo,
		LocalGenerics:     context.LocalGenerics,
		LocalGenericTypes: context.LocalGenericTypes,
	}

	for _, handler := range t.handlers {
		if handler.CanHandle(compLit.Type) {
			isExternal := handler.IsExternal(compLit.Type, instContext)
			if isExternal {
				return
			}

			isExplicit := handler.IsExplicit()

			if !isExplicit {
				if !t.hasInstance(compLit.Type, instContext) {
					return
				}
			}

			if isExplicit {
				counters.GenericTypeInstantiationExplicit++
			} else {
				counters.GenericTypeInstantiationInferred++
			}
			return
		}
	}
}

func (t *GenericTypeCompositeLitCheck) hasInstance(expr ast.Expr, context *InstantiationContext) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		_, hasInstance := context.TypeInfo.Instances[ident]
		return hasInstance
	}
	return false
}

// InstantiationDiversityCheck collects concrete type-argument combinations for locally-defined
// generic structs and functions. Results are written into context.Instantiations so
// they can be read by astAnalyzer after the check run.
type InstantiationDiversityCheck struct{}

func (c *InstantiationDiversityCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.Ident)
	return ok && context.TypeInfo != nil
}

func (c *InstantiationDiversityCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	ident := node.(*ast.Ident)
	instance, hasInstance := context.TypeInfo.Instances[ident]
	if !hasInstance || instance.TypeArgs == nil || instance.TypeArgs.Len() == 0 {
		return
	}

	var kind model.InstantiationKind

	// Check for locally-defined generic struct
	if context.LocalGenericTypes[ident.Name] {
		if named, ok := instance.Type.(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				kind = model.KindStruct
			}
		}
	}

	// Check for locally-defined generic function
	if kind == "" && context.LocalGenerics[ident.Name] != nil {
		if _, isFunc := instance.Type.(*types.Signature); isFunc {
			kind = model.KindFunction
		}
	}

	if kind == "" {
		return
	}

	// Build type-arg string and detect parametric arguments
	isParametric := false
	typeArgStrings := make([]string, instance.TypeArgs.Len())
	for i := 0; i < instance.TypeArgs.Len(); i++ {
		arg := instance.TypeArgs.At(i)
		typeArgStrings[i] = shortTypeName(arg.String())
		if _, ok := arg.(*types.TypeParam); ok {
			isParametric = true
		}
	}
	combo := strings.Join(typeArgStrings, ", ")

	if context.Instantiations == nil {
		context.Instantiations = make(model.InstantiationData)
	}
	if context.Instantiations[ident.Name] == nil {
		context.Instantiations[ident.Name] = make(map[string]model.InstantiationEntry)
	}
	context.Instantiations[ident.Name][combo] = model.InstantiationEntry{
		TypeArgs:     combo,
		IsParametric: isParametric,
		Kind:         kind,
	}
}

// GenericTypeCallCheck counts generic type instantiations that occur via function calls (CallExpr).
// This catches constructor-style patterns where the return type is a generic named type.
type GenericTypeCallCheck struct{}

func (t *GenericTypeCallCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.CallExpr)
	return ok && context.TypeInfo != nil
}

func (t *GenericTypeCallCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	callExpr := node.(*ast.CallExpr)

	if typeAndValue, ok := context.TypeInfo.Types[callExpr]; ok {
		if named, ok := typeAndValue.Type.(*types.Named); ok {
			if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
				typeName := named.Obj().Name()
				if context.LocalGenericTypes[typeName] {
					counters.GenericTypeInstantiationInferred++
				}
			}
		}
	}
}