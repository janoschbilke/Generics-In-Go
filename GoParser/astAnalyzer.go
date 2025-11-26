package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type GenericCounters struct {
	// Funktionen
	FuncTotal   int
	FuncGeneric int

	// Methoden
	MethodTotal               int
	MethodWithGenericReceiver int

	// Structs
	StructTotal        int
	StructGeneric      int
	StructGenericBound int

	// Sonstiges
	TypeDecl        int
	GenericTypeDecl int
	GenericTypeSet  int
}

func analyzeFile(src string) (GenericCounters, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return GenericCounters{}, err
	}

	counters := GenericCounters{}

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
					if _, ok := receiverType.(*ast.IndexExpr); ok {
						counters.MethodWithGenericReceiver++
					} else if _, ok := receiverType.(*ast.IndexListExpr); ok {
						counters.MethodWithGenericReceiver++
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
					// prüfen, ob Constraints != "any" vorkommen
					for _, tp := range node.TypeParams.List {
						if tp.Type != nil {
							isTrivialConstraint := false

							// Check for "any"
							if ident, ok := tp.Type.(*ast.Ident); ok && ident.Name == "any" {
								isTrivialConstraint = true
							}

							// Check for empty interface{}
							if iface, ok := tp.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
								isTrivialConstraint = true
							}

							// Semantic Check: Is the constrant an empty interface defined elsewhere? 
							// E.g. type MyInterface interface{}
							// type Foo3[T MyInterface] struct {
							// 		val T
							// } 
							if ident, ok := tp.Type.(*ast.Ident); ok {
								obj := ident.Obj
								if obj != nil {
									if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
										if iface, ok := ts.Type.(*ast.InterfaceType); ok && iface.Methods != nil && iface.Methods.NumFields() == 0 {
											isTrivialConstraint = true
										}
									}
								}
							}

							if !isTrivialConstraint {
								counters.StructGenericBound++
								break
							}
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
