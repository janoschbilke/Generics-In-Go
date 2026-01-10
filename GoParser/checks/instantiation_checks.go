package checks

import (
	"GoParser/model"
	"go/ast"
	"go/types"
)

type FunctionInstantiationCheck struct {
	handlers []ExpressionHandler
}

func NewFunctionInstantiationCheck() *FunctionInstantiationCheck {
	return &FunctionInstantiationCheck{
		handlers: []ExpressionHandler{
			&IndexExprHandler{},
			&IndexListExprHandler{},
			&InferredIdentHandler{},
			&InferredSelectorHandler{},
		},
	}
}

func (f *FunctionInstantiationCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.CallExpr)
	return ok && context.TypeInfo != nil
}

func (f *FunctionInstantiationCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
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

func (f *FunctionInstantiationCheck) hasInstance(expr ast.Expr, context *InstantiationContext) bool {
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

type TypeInstantiationCheck struct {
	handlers []ExpressionHandler
}

func NewTypeInstantiationCheck() *TypeInstantiationCheck {
	return &TypeInstantiationCheck{
		handlers: []ExpressionHandler{
			&IndexExprHandler{},
			&IndexListExprHandler{},
			&InferredIdentHandler{},
		},
	}
}

func (t *TypeInstantiationCheck) Check(node ast.Node, context *AnalysisContext) bool {
	compLit, ok := node.(*ast.CompositeLit)
	return ok && compLit.Type != nil && context.TypeInfo != nil
}

func (t *TypeInstantiationCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
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

func (t *TypeInstantiationCheck) hasInstance(expr ast.Expr, context *InstantiationContext) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		_, hasInstance := context.TypeInfo.Instances[ident]
		return hasInstance
	}
	return false
}

type TypeInstantiationFromCallCheck struct{}

func (t *TypeInstantiationFromCallCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.CallExpr)
	return ok && context.TypeInfo != nil
}

func (t *TypeInstantiationFromCallCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
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
