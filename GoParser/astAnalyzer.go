package main

import (
	"GoParser/model"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

type ASTAnalyzer interface {
	AnalyzeFile(src string) (model.GenericCounters, error)
	AnalyzeFileWithConfig(src string, enableTypeInference bool) (model.GenericCounters, error)
}

type astAnalyzerImpl struct{}

func NewASTAnalyzer() ASTAnalyzer {
	return &astAnalyzerImpl{}
}

func (a *astAnalyzerImpl) AnalyzeFile(src string) (model.GenericCounters, error) {
	return a.AnalyzeFileWithConfig(src, false)
}

func (a *astAnalyzerImpl) AnalyzeFileWithConfig(src string, enableTypeInference bool) (model.GenericCounters, error) {
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

	// Erweiterung 4: Analyze generic instantiations (only if enabled)
	if enableTypeInference {
		instantiationCounters, err := analyzeGenericInstantiations(file, fset, src)
		if err != nil {
			// Log the error but don't fail completely
			log.Printf("Type inference analysis failed: %v", err)
		} else {
			// Add instantiation counts to main counters
			counters.GenericFuncInstantiationExplicit = instantiationCounters.FuncExplicit
			counters.GenericFuncInstantiationInferred = instantiationCounters.FuncInferred
			counters.GenericFuncInstantiationExternalExplicit = instantiationCounters.FuncExternalExplicit
			counters.GenericFuncInstantiationExternalInferred = instantiationCounters.FuncExternalInferred
			counters.GenericTypeInstantiationExplicit = instantiationCounters.TypeExplicit
			counters.GenericTypeInstantiationInferred = instantiationCounters.TypeInferred
			counters.GenericTypeInstantiationExternalExplicit = instantiationCounters.TypeExternalExplicit
			counters.GenericTypeInstantiationExternalInferred = instantiationCounters.TypeExternalInferred
		}
	}

	return counters, nil
}

// InstantiationCounters tracks generic function/type instantiations
type InstantiationCounters struct {
	FuncExplicit         int
	FuncInferred         int
	FuncExternalExplicit int
	FuncExternalInferred int
	TypeExplicit         int
	TypeInferred         int
	TypeExternalExplicit int
	TypeExternalInferred int
}

// GenericDefinition represents a generic function or method definition
type GenericDefinition struct {
	name       string
	isMethod   bool
	numTypeParams int
}

// collectLocalGenerics collects all generic function and method definitions in the file
func collectLocalGenerics(file *ast.File) map[string]*GenericDefinition {
	generics := make(map[string]*GenericDefinition)

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Type.TypeParams != nil && len(funcDecl.Type.TypeParams.List) > 0 {
				def := &GenericDefinition{
					name:       funcDecl.Name.Name,
					isMethod:   funcDecl.Recv != nil,
					numTypeParams: len(funcDecl.Type.TypeParams.List),
				}
				generics[funcDecl.Name.Name] = def
			}
		}
		return true
	})

	return generics
}

// analyzeGenericInstantiations performs type-based analysis to detect generic instantiations
func analyzeGenericInstantiations(file *ast.File, fset *token.FileSet, src string) (*InstantiationCounters, error) {
	counters := &InstantiationCounters{}
	
	// Collect local generic definitions
	localGenerics := collectLocalGenerics(file)

	// Setup type checker
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			// Ignore type errors during analysis
			// We still want to count what we can
		},
	}
	
	info := &types.Info{
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}

	// Try to type check the file
	_, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, info)
	if err != nil {
		// Even if type checking fails partially, we can still analyze what we have
		log.Printf("Type checking encountered errors (continuing anyway): %v", err)
	}

	// Collect local generic types
	localGenericTypes := collectLocalGenericTypes(file)

	// Analyze all function calls and composite literals
	ast.Inspect(file, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			analyzeCallExpr(callExpr, info, localGenerics, counters)
			// Also check if the call returns a generic type (for inferred instantiations)
			analyzeCallExprForTypeInstantiation(callExpr, info, localGenericTypes, counters)
		}
		// Analyze composite literals for type instantiations: Box[int]{}, Box{value: 1}
		if compLit, ok := n.(*ast.CompositeLit); ok {
			analyzeCompositeLit(compLit, info, localGenericTypes, counters)
		}
		return true
	})

	return counters, nil
}

// collectLocalGenericTypes collects all generic type definitions in the file
func collectLocalGenericTypes(file *ast.File) map[string]bool {
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

// analyzeCompositeLit analyzes composite literals for generic type instantiations
func analyzeCompositeLit(compLit *ast.CompositeLit, info *types.Info, localGenericTypes map[string]bool, counters *InstantiationCounters) {
	// Case 1: Explicit type instantiation - Box[int]{}
	if indexExpr, ok := compLit.Type.(*ast.IndexExpr); ok {
		handleExplicitTypeInstantiation(indexExpr.X, localGenericTypes, counters)
		return
	}
	
	if indexListExpr, ok := compLit.Type.(*ast.IndexListExpr); ok {
		handleExplicitTypeInstantiation(indexListExpr.X, localGenericTypes, counters)
		return
	}

	// Case 2: Inferred type instantiation - Box{value: 1}
	// This needs type info to determine if Box is generic
	if ident, ok := compLit.Type.(*ast.Ident); ok {
		handleInferredTypeInstantiation(ident, info, localGenericTypes, counters)
	}
}

// handleExplicitTypeInstantiation processes explicit type instantiations like Box[int]{}
func handleExplicitTypeInstantiation(typeExpr ast.Expr, localGenericTypes map[string]bool, counters *InstantiationCounters) {
	var typeName string

	if ident, ok := typeExpr.(*ast.Ident); ok {
		typeName = ident.Name
	} else if selExpr, ok := typeExpr.(*ast.SelectorExpr); ok {
		// External type: pkg.Type[int]{}
		typeName = selExpr.Sel.Name
		counters.TypeExternalExplicit++
		return
	} else {
		return
	}

	// Check if it's a local generic type
	if localGenericTypes[typeName] {
		counters.TypeExplicit++
	} else {
		counters.TypeExternalExplicit++
	}
}

// handleInferredTypeInstantiation processes inferred type instantiations like Box{value: 1}
func handleInferredTypeInstantiation(ident *ast.Ident, info *types.Info, localGenericTypes map[string]bool, counters *InstantiationCounters) {
	// Check if this identifier represents a generic instantiation
	if _, hasInstance := info.Instances[ident]; hasInstance {
		// This is an inferred generic type instantiation
		if localGenericTypes[ident.Name] {
			counters.TypeInferred++
		} else {
			counters.TypeExternalInferred++
		}
	}
}

// analyzeCallExprForTypeInstantiation checks if a call expression returns a generic type instance
func analyzeCallExprForTypeInstantiation(callExpr *ast.CallExpr, info *types.Info, localGenericTypes map[string]bool, counters *InstantiationCounters) {
	// Get the type of the call expression result
	if typeAndValue, ok := info.Types[callExpr]; ok {
		// Check if the result type is a Named type (could be generic)
		if named, ok := typeAndValue.Type.(*types.Named); ok {
			// Check if this named type is generic (has type arguments)
			if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
				// This is an instantiated generic type
				typeName := named.Obj().Name()
				
				// Check if it's local or external
				if localGenericTypes[typeName] {
					counters.TypeInferred++
				} else {
					counters.TypeExternalInferred++
				}
			}
		}
	}
}

// analyzeCallExpr analyzes a call expression for generic instantiation
func analyzeCallExpr(callExpr *ast.CallExpr, info *types.Info, localGenerics map[string]*GenericDefinition, counters *InstantiationCounters) {
	// Case 1: Explicit instantiation - f[int](x) or obj.m[int](x)
	// This is represented as IndexExpr or IndexListExpr in Fun
	if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
		handleExplicitInstantiation(indexExpr.X, localGenerics, counters)
		return
	}
	
	if indexListExpr, ok := callExpr.Fun.(*ast.IndexListExpr); ok {
		handleExplicitInstantiation(indexListExpr.X, localGenerics, counters)
		return
	}

	// Case 2: Inferred instantiation - f(x) or obj.m(x)
	// Need to check if the called function is generic using type info
	handleInferredInstantiation(callExpr.Fun, info, localGenerics, counters)
}

// handleExplicitInstantiation processes explicit function instantiations like f[int](x)
func handleExplicitInstantiation(fun ast.Expr, localGenerics map[string]*GenericDefinition, counters *InstantiationCounters) {
	var funcName string
	isExternal := false

	// Check if it's a method/selector call (external package)
	if selExpr, ok := fun.(*ast.SelectorExpr); ok {
		funcName = selExpr.Sel.Name
		isExternal = true
	} else if ident, ok := fun.(*ast.Ident); ok {
		// Direct function call
		funcName = ident.Name
	} else {
		return
	}

	// Check if it's a local generic function
	if _, exists := localGenerics[funcName]; exists && !isExternal {
		counters.FuncExplicit++
	} else {
		// External generic function
		counters.FuncExternalExplicit++
	}
}

// handleInferredInstantiation processes inferred function instantiations like f(x)
func handleInferredInstantiation(fun ast.Expr, info *types.Info, localGenerics map[string]*GenericDefinition, counters *InstantiationCounters) {
	var funcName string
	var ident *ast.Ident
	isExternal := false

	// Check if it's a selector call (external package)
	if selExpr, ok := fun.(*ast.SelectorExpr); ok {
		funcName = selExpr.Sel.Name
		ident = selExpr.Sel
		isExternal = true
	} else if identExpr, ok := fun.(*ast.Ident); ok {
		// Direct function call
		funcName = identExpr.Name
		ident = identExpr
	} else {
		return
	}

	// Check if this identifier represents a generic instantiation
	// types.Info.Instances contains instantiated generics
	if _, hasInstance := info.Instances[ident]; hasInstance {
		// This is an inferred generic instantiation
		if _, exists := localGenerics[funcName]; exists && !isExternal {
			counters.FuncInferred++
		} else {
			// External generic with inference
			counters.FuncExternalInferred++
		}
	}
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
