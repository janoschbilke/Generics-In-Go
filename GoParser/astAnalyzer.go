package main

import (
	"GoParser/checks"
	"GoParser/model"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
)

type ASTAnalyzer interface {
	AnalyzeFileWithConfig(src string, enableTypeInference bool) (model.GenericCounters, error)
}

type astAnalyzerImpl struct{}

func NewASTAnalyzer() ASTAnalyzer {
	return &astAnalyzerImpl{}
}

func (a *astAnalyzerImpl) AnalyzeFileWithConfig(src string, enableTypeInference bool) (model.GenericCounters, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return model.GenericCounters{}, err
	}

	typeBoundsInfo := checks.CollectTypeBoundsInfo(file)
	localGenerics := checks.CollectLocalGenerics(file)
	localGenericTypes := checks.CollectLocalGenericTypes(file)

	context := &checks.AnalysisContext{
		TypeBoundsInfo:    typeBoundsInfo,
		LocalGenerics:     localGenerics,
		LocalGenericTypes: localGenericTypes,
	}

	basicChecks := []checks.ASTCheck{
		&checks.FunctionCheck{},
		&checks.MethodCheck{},
		&checks.TypeDeclCheck{},
		&checks.StructCheck{},
		&checks.TypeSetCheck{},
	}

	counters := model.GenericCounters{}
	runner := checks.NewCheckRunner(basicChecks)
	runner.RunChecks(file, &counters, context)

	if enableTypeInference {
		typeInfo, err := performTypeChecking(file, fset, src)
		if err != nil {
			log.Printf("Type inference analysis failed: %v", err)
		} else {
			context.TypeInfo = typeInfo
			instantiationChecks := []checks.ASTCheck{
				checks.NewFunctionInstantiationCheck(),
				checks.NewTypeInstantiationCheck(),
				&checks.TypeInstantiationFromCallCheck{},
			}
			instRunner := checks.NewCheckRunner(instantiationChecks)
			instRunner.RunChecks(file, &counters, context)
		}
	}

	return counters, nil
}

func performTypeChecking(file *ast.File, fset *token.FileSet, src string) (*types.Info, error) {
	conf := types.Config{
		Importer: nil, // Explicit choice to not use importer and avoid external dependencies
		Error:    func(err error) {},
	}

	info := &types.Info{
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}

	_, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, info)
	if err != nil {
		log.Printf("Type checking encountered errors (continuing anyway): %v", err)
	}

	return info, nil
}
