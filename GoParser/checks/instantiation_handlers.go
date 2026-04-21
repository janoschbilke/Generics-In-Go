package checks

import (
	"go/ast"
	"go/types"
)

type ExpressionHandler interface {
	CanHandle(expr ast.Expr) bool
	IsExplicit() bool
	IsExternal(expr ast.Expr, context *InstantiationContext) bool
}

type InstantiationContext struct {
	TypeInfo          *types.Info
	LocalGenerics     map[string]*GenericDefinition
	LocalGenericTypes map[string]bool
}

type InstantiationResult struct {
	IsLocal    bool
	IsExplicit bool
}

type IndexExprHandler struct{}

func (h *IndexExprHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.IndexExpr)
	return ok
}

func (h *IndexExprHandler) IsExplicit() bool {
	return true
}

func (h *IndexExprHandler) IsExternal(expr ast.Expr, context *InstantiationContext) bool {
	indexExpr := expr.(*ast.IndexExpr)
	if _, ok := indexExpr.X.(*ast.SelectorExpr); ok {
		return true
	}
	if ident, ok := indexExpr.X.(*ast.Ident); ok {
		_, existsInFuncs := context.LocalGenerics[ident.Name]
		_, existsInTypes := context.LocalGenericTypes[ident.Name]
		return !existsInFuncs && !existsInTypes
	}
	return false
}

type IndexListExprHandler struct{}

func (h *IndexListExprHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.IndexListExpr)
	return ok
}

func (h *IndexListExprHandler) IsExplicit() bool {
	return true
}

func (h *IndexListExprHandler) IsExternal(expr ast.Expr, context *InstantiationContext) bool {
	indexListExpr := expr.(*ast.IndexListExpr)
	if _, ok := indexListExpr.X.(*ast.SelectorExpr); ok {
		return true
	}
	if ident, ok := indexListExpr.X.(*ast.Ident); ok {
		_, existsInFuncs := context.LocalGenerics[ident.Name]
		_, existsInTypes := context.LocalGenericTypes[ident.Name]
		return !existsInFuncs && !existsInTypes
	}
	return false
}

type InferredIdentHandler struct{}

func (h *InferredIdentHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.Ident)
	return ok
}

func (h *InferredIdentHandler) IsExplicit() bool {
	return false
}

func (h *InferredIdentHandler) IsExternal(expr ast.Expr, context *InstantiationContext) bool {
	ident := expr.(*ast.Ident)
	_, existsInFuncs := context.LocalGenerics[ident.Name]
	_, existsInTypes := context.LocalGenericTypes[ident.Name]
	return !existsInFuncs && !existsInTypes
}

func (h *InferredIdentHandler) HasInstance(ident *ast.Ident, context *InstantiationContext) bool {
	_, hasInstance := context.TypeInfo.Instances[ident]
	return hasInstance
}

type InferredSelectorHandler struct{}

func (h *InferredSelectorHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.SelectorExpr)
	return ok
}

func (h *InferredSelectorHandler) IsExplicit() bool {
	return false
}

func (h *InferredSelectorHandler) IsExternal(expr ast.Expr, context *InstantiationContext) bool {
	return true
}

func (h *InferredSelectorHandler) HasInstance(selExpr *ast.SelectorExpr, context *InstantiationContext) bool {
	_, hasInstance := context.TypeInfo.Instances[selExpr.Sel]
	return hasInstance
}
