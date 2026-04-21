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

func CollectTypeBoundsInfo(file *ast.File) map[string]TypeBoundInfo {
	typeBoundsInfo := make(map[string]TypeBoundInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
				info := TypeBoundInfo{}

				for _, tp := range typeSpec.TypeParams.List {
					if tp.Type != nil {
						isTrivial := false

						if ident, ok := tp.Type.(*ast.Ident); ok && ident.Name == "any" {
							isTrivial = true
						}

						if iface, ok := tp.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
							isTrivial = true
						}

						if ident, ok := tp.Type.(*ast.Ident); ok {
							obj := ident.Obj
							if obj != nil {
								if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
									if iface, ok := ts.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
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
				}
				typeBoundsInfo[typeSpec.Name.Name] = info
			}
		}
		return true
	})

	return typeBoundsInfo
}

func CollectLocalGenerics(file *ast.File) map[string]*GenericDefinition {
	generics := make(map[string]*GenericDefinition)

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Type.TypeParams != nil && len(funcDecl.Type.TypeParams.List) > 0 {
				def := &GenericDefinition{
					Name:          funcDecl.Name.Name,
					IsMethod:      funcDecl.Recv != nil,
					NumTypeParams: len(funcDecl.Type.TypeParams.List),
				}
				generics[funcDecl.Name.Name] = def
			}
		}
		return true
	})

	return generics
}

func CollectLocalGenericTypes(file *ast.File) map[string]bool {
	genericTypes := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
				genericTypes[typeSpec.Name.Name] = true
			}
		}
		return true
	})

	return genericTypes
}
