package checks

import (
	"GoParser/model"
	"go/ast"
)

type TypeDeclCheck struct{}

func (t *TypeDeclCheck) Check(node ast.Node, context *AnalysisContext) bool {
	_, ok := node.(*ast.TypeSpec)
	return ok
}

func (t *TypeDeclCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	typeSpec := node.(*ast.TypeSpec)
	counters.TypeDecl++
	if typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
		counters.GenericTypeDecl++
	}
}

type StructCheck struct{}

func (s *StructCheck) Check(node ast.Node, context *AnalysisContext) bool {
	typeSpec, ok := node.(*ast.TypeSpec)
	if !ok {
		return false
	}
	_, isStruct := typeSpec.Type.(*ast.StructType)
	return isStruct
}

func (s *StructCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	typeSpec := node.(*ast.TypeSpec)
	counters.StructTotal++

	if typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
		counters.StructGeneric++

		if info, exists := context.TypeBoundsInfo[typeSpec.Name.Name]; exists {
			if info.HasNonTrivialBound {
				counters.StructGenericBound++
			}

			if info.HasStructBound {
				counters.StructAsTypeBound++
			}
		}
	}
}

type TypeSetCheck struct{}

func (t *TypeSetCheck) Check(node ast.Node, context *AnalysisContext) bool {
	typeSpec, ok := node.(*ast.TypeSpec)
	if !ok {
		return false
	}
	iface, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok || iface.Methods == nil {
		return false
	}
	for _, field := range iface.Methods.List {
		if _, isBinary := field.Type.(*ast.BinaryExpr); isBinary {
			return true
		}
	}
	return false
}

func (t *TypeSetCheck) Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext) {
	counters.GenericTypeSet++
}
