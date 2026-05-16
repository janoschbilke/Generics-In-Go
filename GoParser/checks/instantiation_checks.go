package checks

import (
	"GoParser/model"
	"go/ast"
	"go/types"
	"strings"
)

// Counts generic function call sites (CallExpr).

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
		TypeInfo:                 context.TypeInfo,
		LocalGenerics:            context.LocalGenerics,
		LocalGenericTypes:        context.LocalGenericTypes,
		ProjectLocalGenerics:     context.ProjectLocalGenerics,
		ProjectLocalGenericTypes: context.ProjectLocalGenericTypes,
		ProjectImportPaths:       context.ProjectImportPaths,
	}

	for _, handler := range f.handlers {
		if !handler.CanHandle(callExpr.Fun) {
			continue
		}
		if handler.IsExternal(callExpr.Fun, instContext) {
			return
		}

		isExplicit := handler.IsExplicit()
		isMethod := isMethodCallOnGenericType(callExpr.Fun, instContext)

		if !isExplicit {
			// Methods on generic types do NOT appear in types.Info.Instances
			// (they have no own TypeParams); skip the instance check for them.
			if !isMethod {
				if !f.hasInstance(callExpr.Fun, instContext) {
					return
				}
			}
		}

		if isMethod {
			if isExplicit {
				counters.GenericMethodInstantiationExplicit++
			} else {
				counters.GenericMethodInstantiationInferred++
			}
		} else {
			if isExplicit {
				counters.GenericFuncInstantiationExplicit++
			} else {
				counters.GenericFuncInstantiationInferred++
			}
		}
		return
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

func isMethodCallOnGenericType(expr ast.Expr, ctx *InstantiationContext) bool {
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok || ctx.TypeInfo == nil {
		return false
	}
	obj, ok := ctx.TypeInfo.Uses[selExpr.Sel]
	if !ok {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	return ok && sig.Recv() != nil
}

// Counts generic type instantiations via composite literals (CompositeLit).

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
		TypeInfo:                 context.TypeInfo,
		LocalGenerics:            context.LocalGenerics,
		LocalGenericTypes:        context.LocalGenericTypes,
		ProjectLocalGenerics:     context.ProjectLocalGenerics,
		ProjectLocalGenericTypes: context.ProjectLocalGenericTypes,
		ProjectImportPaths:       context.ProjectImportPaths,
	}

	for _, handler := range t.handlers {
		if !handler.CanHandle(compLit.Type) {
			continue
		}
		if handler.IsExternal(compLit.Type, instContext) {
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

func (t *GenericTypeCompositeLitCheck) hasInstance(expr ast.Expr, context *InstantiationContext) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		_, hasInstance := context.TypeInfo.Instances[ident]
		return hasInstance
	}
	return false
}


// Handles two node types:
//   - *ast.Ident  → structs and free generic functions
//   - *ast.SelectorExpr → methods on generic types

type InstantiationDiversityCheck struct{}

func (c *InstantiationDiversityCheck) Check(node ast.Node, context *AnalysisContext) bool {
	if context.TypeInfo == nil {
		return false
	}
	switch node.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return true
	}
	return false
}

func (c *InstantiationDiversityCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	switch n := node.(type) {
	case *ast.Ident:
		c.updateIdent(n, context)
	case *ast.SelectorExpr:
		c.updateSelectorExpr(n, context)
	}
}

// updateIdent handles structs and free generic functions via types.Info.Instances.
func (c *InstantiationDiversityCheck) updateIdent(ident *ast.Ident, context *AnalysisContext) {
	instance, hasInstance := context.TypeInfo.Instances[ident]
	if !hasInstance || instance.TypeArgs == nil || instance.TypeArgs.Len() == 0 {
		return
	}

	var kind model.InstantiationKind

	if context.LocalGenericTypes[ident.Name] {
		if named, ok := instance.Type.(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				kind = model.KindStruct
			}
		}
	}

	if kind == "" && context.LocalGenerics[ident.Name] != nil {
		if _, isFunc := instance.Type.(*types.Signature); isFunc {
			kind = model.KindFunction
		}
	}

	if kind == "" {
		return
	}

	c.record(context, kind, ident.Name, instance.TypeArgs)
}

func (c *InstantiationDiversityCheck) updateSelectorExpr(selExpr *ast.SelectorExpr, context *AnalysisContext) {
	// Must be a method on a local generic type
	obj, ok := context.TypeInfo.Uses[selExpr.Sel]
	if !ok {
		return
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return
	}

	// Get the concrete receiver type from the X expression
	xTV, ok := context.TypeInfo.Types[selExpr.X]
	if !ok {
		return
	}
	recvType := xTV.Type
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok {
		return
	}

	// Only count if the receiver type is locally-defined or project-internal
	typeName := named.Obj().Name()
	isLocal := context.LocalGenericTypes[typeName]
	isProjectInternal := named.Obj().Pkg() != nil &&
		context.ProjectImportPaths != nil &&
		context.ProjectImportPaths[named.Obj().Pkg().Path()]
	if !isLocal && !isProjectInternal {
		return
	}

	typeArgs := named.TypeArgs()
	if typeArgs == nil || typeArgs.Len() == 0 {
		return
	}

	// Build type-arg string
	isParametric := false
	typeArgStrings := make([]string, typeArgs.Len())
	for i := 0; i < typeArgs.Len(); i++ {
		arg := typeArgs.At(i)
		typeArgStrings[i] = unqualifiedTypeName(arg)
		if _, ok := arg.(*types.TypeParam); ok {
			isParametric = true
		}
	}
	combo := strings.Join(typeArgStrings, ", ")

	key := model.InstantiationKey(model.KindMethod, typeName+"."+selExpr.Sel.Name)

	if context.Instantiations == nil {
		context.Instantiations = make(model.InstantiationData)
	}
	if context.Instantiations[key] == nil {
		context.Instantiations[key] = make(map[string]model.InstantiationEntry)
	}
	context.Instantiations[key][combo] = model.InstantiationEntry{
		Name:         selExpr.Sel.Name,
		TypeArgs:     combo,
		IsParametric: isParametric,
		Kind:         model.KindMethod,
	}
}

func (c *InstantiationDiversityCheck) record(context *AnalysisContext, kind model.InstantiationKind, name string, typeArgs *types.TypeList) {
	isParametric := false
	typeArgStrings := make([]string, typeArgs.Len())
	for i := 0; i < typeArgs.Len(); i++ {
		arg := typeArgs.At(i)
		typeArgStrings[i] = unqualifiedTypeName(arg)
		if _, ok := arg.(*types.TypeParam); ok {
			isParametric = true
		}
	}
	combo := strings.Join(typeArgStrings, ", ")
	key := model.InstantiationKey(kind, name)

	if context.Instantiations == nil {
		context.Instantiations = make(model.InstantiationData)
	}
	if context.Instantiations[key] == nil {
		context.Instantiations[key] = make(map[string]model.InstantiationEntry)
	}
	context.Instantiations[key][combo] = model.InstantiationEntry{
		Name:         name,
		TypeArgs:     combo,
		IsParametric: isParametric,
		Kind:         kind,
	}
}

// Counts generic type instantiations that occur via function calls (CallExpr).

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
