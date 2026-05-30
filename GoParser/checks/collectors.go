package checks

import (
	"go/ast"
	"go/types"
)

func unqualifiedTypeName(t types.Type) string {
	return types.TypeString(t, func(_ *types.Package) string {
		return "" // omit package path; TypeString keeps the bare type name
	})
}

func CollectAll(file *ast.File) (map[string]TypeBoundInfo, map[string]*GenericDefinition, map[string]bool) {
	typeBoundsInfo := make(map[string]TypeBoundInfo)
	localGenerics := make(map[string]*GenericDefinition)
	localGenericTypes := make(map[string]bool)

	type pendingMethod struct {
		methodName       string
		receiverTypeName string
	}
	var pendingMethods []pendingMethod

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {

		case *ast.TypeSpec:
			if node.TypeParams == nil || len(node.TypeParams.List) == 0 {
				return true
			}
			info := TypeBoundInfo{}
			for _, tp := range node.TypeParams.List {
				if tp.Type == nil {
					continue
				}
				isTrivial := false

				if ident, ok := tp.Type.(*ast.Ident); ok && ident.Name == "any" {
					isTrivial = true
				}

				if iface, ok := tp.Type.(*ast.InterfaceType); ok &&
					iface.Methods != nil && iface.Methods.NumFields() == 0 {
					isTrivial = true
				}

				if ident, ok := tp.Type.(*ast.Ident); ok {
					if obj := ident.Obj; obj != nil {
						if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
							if iface, ok := ts.Type.(*ast.InterfaceType); ok &&
								iface.Methods != nil && iface.Methods.NumFields() == 0 {
								isTrivial = true
							}
							if _, ok := ts.Type.(*ast.StructType); ok {
								info.HasStructBound = true
							}
						}
					}
				}
				if !isTrivial {
					info.HasNonTrivialBound = true
				}
			}
			typeBoundsInfo[node.Name.Name] = info

			localGenericTypes[node.Name.Name] = true

		case *ast.FuncDecl:
			if node.Type.TypeParams != nil && len(node.Type.TypeParams.List) > 0 {

				localGenerics[node.Name.Name] = &GenericDefinition{
					Name:          node.Name.Name,
					IsMethod:      node.Recv != nil,
					NumTypeParams: len(node.Type.TypeParams.List),
				}
			} else if node.Recv != nil {

				if recvTypeName := extractGenericReceiverTypeName(node.Recv); recvTypeName != "" {
					pendingMethods = append(pendingMethods, pendingMethod{
						methodName:       node.Name.Name,
						receiverTypeName: recvTypeName,
					})
				}
			}
		}
		return true
	})

	for _, pm := range pendingMethods {
		if localGenericTypes[pm.receiverTypeName] {
			key := pm.receiverTypeName + "." + pm.methodName
			localGenerics[key] = &GenericDefinition{
				Name:             pm.methodName,
				IsMethod:         true,
				NumTypeParams:    0,
				ReceiverTypeName: pm.receiverTypeName,
			}
		}
	}

	return typeBoundsInfo, localGenerics, localGenericTypes
}

func extractGenericReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	recvType := recv.List[0].Type

	if star, ok := recvType.(*ast.StarExpr); ok {
		recvType = star.X
	}

	switch t := recvType.(type) {
	case *ast.IndexExpr: // single type param: Stack[T]
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.IndexListExpr: // multiple type params: Pair[A, B]
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
