package main

import (
	"GoParser/model"
	"go/ast"
	"go/parser"
	"go/token"
)

type ASTAnalyzer interface {
	AnalyzeFile(src string) (model.GenericCounters, error)
}

type astAnalyzerImpl struct{}

func NewASTAnalyzer() ASTAnalyzer {
	return &astAnalyzerImpl{}
}

func (a *astAnalyzerImpl) AnalyzeFile(src string) (model.GenericCounters, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return model.GenericCounters{}, err
	}

	// First pass: collect type bounds information (for Erweiterung 2 & 3)
	typeBoundsInfo := collectTypeBoundsInfo(file)

	// Second pass: analyze file with information about type bounds available
	counters, err := analyzeASTAndGetCounters(file, typeBoundsInfo)
	if err != nil {
		return model.GenericCounters{}, err
	}

	return counters, nil
}

// TypeBoundInfo stores information about a type's bounds
type TypeBoundInfo struct {
	hasNonTrivialBound bool
	hasStructBound     bool // Erweiterung 2: tracks if any bound is a struct
}

func collectTypeBoundsInfo(file *ast.File) map[string]TypeBoundInfo {
	typeBoundsInfo := make(map[string]TypeBoundInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if typeSpec.TypeParams != nil && len(typeSpec.TypeParams.List) > 0 {
				info := TypeBoundInfo{}

				for _, tp := range typeSpec.TypeParams.List {
					if tp.Type != nil {
						isTrivial := false

						// Check for "any"
						if ident, ok := tp.Type.(*ast.Ident); ok && ident.Name == "any" {
							isTrivial = true
						}

						// Check for empty interface{}
						if iface, ok := tp.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
							isTrivial = true
						}

						// Check if constraint is an empty interface or struct defined elsewhere
						if ident, ok := tp.Type.(*ast.Ident); ok {
							obj := ident.Obj
							if obj != nil {
								if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
									// Erweiterung 1: Check if constraint is an empty interface
									if iface, ok := ts.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
										isTrivial = true
									}
									// Erweiterung 2: Check if constraint is a struct type
									if _, ok := ts.Type.(*ast.StructType); ok {
										info.hasStructBound = true
									}
								}
							}
						}

						if !isTrivial {
							info.hasNonTrivialBound = true
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

func analyzeASTAndGetCounters(file *ast.File, typeBoundsInfo map[string]TypeBoundInfo) (model.GenericCounters, error) {
	counters := model.GenericCounters{}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {

		// Funktionen und Methoden
		case *ast.FuncDecl:
			if node.Recv == nil {
				// Normale Funktion
				counters.FuncTotal++
				if node.Type.TypeParams != nil && len(node.Type.TypeParams.List) > 0 {
					counters.FuncGeneric++
				}
			}
			if node.Recv != nil {
				counters.MethodTotal++

				if node.Recv.List != nil {
					// 1. Zuerst den Typ des Receivers abrufen.
					receiverType := node.Recv.List[0].Type

					// 2. Prüfen, ob der Receiver ein Pointer ist.
					if starExpr, ok := receiverType.(*ast.StarExpr); ok {
						receiverType = starExpr.X // Wenn ja, zum eigentlichen Typ wechseln.
					}

					// 3. Nun prüfen, ob der Typ des Receivers generisch ist.
					// Ein generischer Typ wird im AST als *ast.IndexExpr (für einen Typparameter) oder *ast.IndexListExpr (für mehrere) dargestellt.
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

						// Erweiterung 3: Check if receiver type has non-trivial bound
						if info, exists := typeBoundsInfo[receiverTypeName]; exists {
							if info.hasNonTrivialBound {
								counters.MethodWithGenericReceiverNonTrivialTypeBound++
							} else {
								counters.MethodWithGenericReceiverTrivialTypeBound++
							}
						}
					}
				}
			}

		// Typ-Deklarationen (Structs, Aliase, Interfaces, ...)
		case *ast.TypeSpec:
			counters.TypeDecl++
			if node.TypeParams != nil && len(node.TypeParams.List) > 0 {
				counters.GenericTypeDecl++
			}

			// Structs zählen
			if _, ok := node.Type.(*ast.StructType); ok {
				counters.StructTotal++
				if node.TypeParams != nil && len(node.TypeParams.List) > 0 {
					counters.StructGeneric++

					// Use collected type bounds info from first pass
					if info, exists := typeBoundsInfo[node.Name.Name]; exists {
						// Erweiterung 1: Count structs with non-trivial bounds
						if info.hasNonTrivialBound {
							counters.StructGenericBound++
						}

						// Erweiterung 2: Count structs that have a struct as type bound
						if info.hasStructBound {
							counters.StructAsTypeBound++
						}
					}
				}
			}

			// Interfaces auf TypeSets prüfen
			iface, ok := node.Type.(*ast.InterfaceType)
			if ok && iface.Methods != nil {
				for _, field := range iface.Methods.List {
					// Ein Type Set im AST ist ein BinaryExpr mit '|' oder '&'
					if _, ok := field.Type.(*ast.BinaryExpr); ok {
						counters.GenericTypeSet++
					}
				}
			}
		}
		return true
	})

	return counters, nil
}
