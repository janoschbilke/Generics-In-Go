package checks

import (
	"GoParser/model"
	"go/ast"
)

type FunctionCheck struct{}

func (f *FunctionCheck) Check(node ast.Node, context *AnalysisContext) bool {
	funcDecl, ok := node.(*ast.FuncDecl)
	return ok && funcDecl.Recv == nil
}

func (f *FunctionCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	funcDecl := node.(*ast.FuncDecl)
	counters.FuncTotal++
	if funcDecl.Type.TypeParams != nil && len(funcDecl.Type.TypeParams.List) > 0 {
		counters.FuncGeneric++
	}
}

type MethodCheck struct{}

func (m *MethodCheck) Check(node ast.Node, context *AnalysisContext) bool {
	funcDecl, ok := node.(*ast.FuncDecl)
	return ok && funcDecl.Recv != nil
}

func (m *MethodCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	funcDecl := node.(*ast.FuncDecl)
	counters.MethodTotal++

	if funcDecl.Recv.List == nil {
		return
	}

	receiverType := funcDecl.Recv.List[0].Type

	if starExpr, ok := receiverType.(*ast.StarExpr); ok {
		receiverType = starExpr.X
	}

	isGenericReceiver := false
	var receiverTypeName string

	if indexExpr, ok := receiverType.(*ast.IndexExpr); ok {
		isGenericReceiver = true
		if ident, ok := indexExpr.X.(*ast.Ident); ok {
			receiverTypeName = ident.Name
		}
	} else if indexListExpr, ok := receiverType.(*ast.IndexListExpr); ok {
		isGenericReceiver = true
		if ident, ok := indexListExpr.X.(*ast.Ident); ok {
			receiverTypeName = ident.Name
		}
	}

	if isGenericReceiver {
		counters.MethodWithGenericReceiver++

		if info, exists := context.TypeBoundsInfo[receiverTypeName]; exists {
			if info.HasNonTrivialBound {
				counters.MethodWithGenericReceiverNonTrivialTypeBound++
			} else {
				counters.MethodWithGenericReceiverTrivialTypeBound++
			}
		}
	}
}
