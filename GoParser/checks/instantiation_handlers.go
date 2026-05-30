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
	TypeInfo                 *types.Info
	LocalGenerics            map[string]*GenericDefinition
	LocalGenericTypes        map[string]bool
	ProjectLocalGenerics     map[string]*GenericDefinition
	ProjectLocalGenericTypes map[string]bool
	ProjectImportPaths       map[string]bool
}

type InstantiationResult struct {
	IsLocal    bool
	IsExplicit bool
}

func isProjectPkg(pkgPath string, ctx *InstantiationContext) bool {
	return ctx.ProjectImportPaths != nil && ctx.ProjectImportPaths[pkgPath]
}

func isLocalGeneric(name string, ctx *InstantiationContext) bool {
	_, inFuncs := ctx.LocalGenerics[name]
	_, inTypes := ctx.LocalGenericTypes[name]
	return inFuncs || inTypes
}

func isProjectGeneric(pkgPath, name string, ctx *InstantiationContext) bool {
	key := pkgPath + "." + name
	_, inFuncs := ctx.ProjectLocalGenerics[key]
	_, inTypes := ctx.ProjectLocalGenericTypes[key]
	return inFuncs || inTypes
}

func resolveIndexExprLocality(x ast.Expr, ctx *InstantiationContext) (isExternal bool) {
	selExpr, isSel := x.(*ast.SelectorExpr)
	if !isSel {
		if ident, ok := x.(*ast.Ident); ok {
			return !isLocalGeneric(ident.Name, ctx)
		}
		return true
	}

	if ctx.TypeInfo != nil {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			if obj, ok := ctx.TypeInfo.Uses[ident]; ok {
				if pkgName, isPkg := obj.(*types.PkgName); isPkg {
					pkgPath := pkgName.Imported().Path()
					if isProjectPkg(pkgPath, ctx) {
						return !isProjectGeneric(pkgPath, selExpr.Sel.Name, ctx)
					}
					return true
				}
			}
		}

		if xTV, ok := ctx.TypeInfo.Types[selExpr.X]; ok {
			recvType := xTV.Type
			if ptr, ok := recvType.(*types.Pointer); ok {
				recvType = ptr.Elem()
			}
			if named, ok := recvType.(*types.Named); ok {
				typeName := named.Obj().Name()
				if ctx.LocalGenericTypes[typeName] {
					return false
				}
				if named.Obj().Pkg() != nil && isProjectPkg(named.Obj().Pkg().Path(), ctx) {
					return false
				}
			}
		}
	}
	return true
}

// Handles explicit single-type-arg instantiations: func[T](...) or Type[T]{...}
type IndexExprHandler struct{}

func (h *IndexExprHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.IndexExpr)
	return ok
}

func (h *IndexExprHandler) IsExplicit() bool { return true }

func (h *IndexExprHandler) IsExternal(expr ast.Expr, ctx *InstantiationContext) bool {
	return resolveIndexExprLocality(expr.(*ast.IndexExpr).X, ctx)
}

// Handles explicit multi-type-arg instantiations: func[T1, T2](...) or Type[T1, T2]{...}
type IndexListExprHandler struct{}

func (h *IndexListExprHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.IndexListExpr)
	return ok
}

func (h *IndexListExprHandler) IsExplicit() bool { return true }

func (h *IndexListExprHandler) IsExternal(expr ast.Expr, ctx *InstantiationContext) bool {
	return resolveIndexExprLocality(expr.(*ast.IndexListExpr).X, ctx)
}

// Handles inferred instantiations: func(...) where the type is inferred.
type InferredIdentHandler struct{}

func (h *InferredIdentHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.Ident)
	return ok
}

func (h *InferredIdentHandler) IsExplicit() bool { return false }

func (h *InferredIdentHandler) IsExternal(expr ast.Expr, ctx *InstantiationContext) bool {
	ident := expr.(*ast.Ident)
	return !isLocalGeneric(ident.Name, ctx)
}

func (h *InferredIdentHandler) HasInstance(ident *ast.Ident, context *InstantiationContext) bool {
	_, hasInstance := context.TypeInfo.Instances[ident]
	return hasInstance
}

// Handles selector expressions: pkg.Func(...) or receiver.Method(...).
type InferredSelectorHandler struct{}

func (h *InferredSelectorHandler) CanHandle(expr ast.Expr) bool {
	_, ok := expr.(*ast.SelectorExpr)
	return ok
}

func (h *InferredSelectorHandler) IsExplicit() bool { return false }

func (h *InferredSelectorHandler) IsExternal(expr ast.Expr, ctx *InstantiationContext) bool {
	if ctx.TypeInfo == nil {
		return true
	}
	selExpr := expr.(*ast.SelectorExpr)

	obj, ok := ctx.TypeInfo.Uses[selExpr.Sel]
	if !ok {
		return true
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return true // not a function
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return true
	}

	// Dereference pointer receiver
	recvType := sig.Recv().Type()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok {
		return true
	}

	// Local package
	if ctx.LocalGenericTypes[named.Obj().Name()] {
		return false
	}
	// Project-internal package
	if named.Obj().Pkg() != nil && isProjectPkg(named.Obj().Pkg().Path(), ctx) {
		return false
	}
	return true
}

func (h *InferredSelectorHandler) HasInstance(selExpr *ast.SelectorExpr, context *InstantiationContext) bool {
	_, hasInstance := context.TypeInfo.Instances[selExpr.Sel]
	return hasInstance
}
