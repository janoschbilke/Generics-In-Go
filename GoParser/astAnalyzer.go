package main

import (
	"GoParser/checks"
	"GoParser/importer"
	"GoParser/model"
	"GoParser/utils"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

type ASTAnalyzer interface {
	AnalyzeProject(files []model.FileInfo, enableTypeInference bool) (model.GenericCounters, model.InstantiationData, error)
}

type astAnalyzerImpl struct{}

func NewASTAnalyzer() ASTAnalyzer {
	return &astAnalyzerImpl{}
}

func (a *astAnalyzerImpl) AnalyzeProject(files []model.FileInfo, enableTypeInference bool) (model.GenericCounters, model.InstantiationData, error) {
	var totalCounters model.GenericCounters
	allInstantiations := make(model.InstantiationData)

	packages := groupByPackage(files)

	if !enableTypeInference {
		for _, pkgFiles := range packages {
			for _, f := range pkgFiles {
				fset := token.NewFileSet()
				astFile, err := parser.ParseFile(fset, f.Path, f.Content, parser.AllErrors)
				if err != nil && astFile == nil {
					continue
				}
				typeBoundsInfo, localGenerics, localGenericTypes := checks.CollectAll(astFile)
				ctx := &checks.AnalysisContext{
					TypeBoundsInfo:    typeBoundsInfo,
					LocalGenerics:     localGenerics,
					LocalGenericTypes: localGenericTypes,
				}
				runner := checks.NewCheckRunner(getBasicChecks())
				runner.RunChecks(astFile, &totalCounters, ctx)
			}
		}
		return totalCounters, allInstantiations, nil
	}

	moduleName := findModuleName(files)
	projectImporter := importer.New()

	parsedFilesToAST := parseFilesToASTPackage(packages, moduleName)
	sortedDirs := utils.TopoSortPackages(parsedFilesToAST, moduleName)

	checkedPackages := extractTypeInfoForPackages(sortedDirs, parsedFilesToAST, projectImporter)

	// Collect project-wide symbols for all packages to enable cross-package checks and store package-specific symbols for efficient access during checks (symbolsByPackage as caching mechanism)
	projectLocalGenerics, projectLocalGenericTypes, projectImportPaths, symbolsByPackage := collectProjectSymbols(checkedPackages)

	for _, cp := range checkedPackages {
		pc := symbolsByPackage[cp.pkg.Dir]

		allChecks := []checks.ASTCheck{}
		allChecks = append(allChecks, getBasicChecks()...)
		allChecks = append(allChecks, getTypeChecks()...)
		runner := checks.NewCheckRunner(allChecks)

		for _, astFile := range cp.pkg.AstFiles {
			ctx := &checks.AnalysisContext{
				TypeBoundsInfo:           pc.typeBoundsInfo,
				LocalGenerics:            pc.localGenerics,
				LocalGenericTypes:        pc.localGenericTypes,
				TypeInfo:                 cp.typeInfo,
				ProjectLocalGenerics:     projectLocalGenerics,
				ProjectLocalGenericTypes: projectLocalGenericTypes,
				ProjectImportPaths:       projectImportPaths,
			}
			runner.RunChecks(astFile, &totalCounters, ctx)
			allInstantiations.Merge(ctx.Instantiations)
		}
	}

	return totalCounters, allInstantiations, nil
}

type checkedPkg struct {
	pkg      *model.ParsedPkg
	typeInfo *types.Info
}

func extractTypeInfoForPackages(sortedDirs []string, parsedFilesToAST map[string]*model.ParsedPkg, projectImporter *importer.ProjectImporter) []checkedPkg {
	var checked []checkedPkg

	for _, dir := range sortedDirs {
		pkg := parsedFilesToAST[dir]
		typeInfo, typedPkg := performTypeChecking(pkg.Fset, pkg.AstFiles, pkg.ImportPath, projectImporter)
		if typedPkg != nil && pkg.ImportPath != "" {
			projectImporter.AddPackage(pkg.ImportPath, typedPkg)
		}
		checked = append(checked, checkedPkg{pkg: pkg, typeInfo: typeInfo})
	}
	return checked
}

type packageSymbols struct {
	localGenerics     map[string]*checks.GenericDefinition
	localGenericTypes map[string]bool
	typeBoundsInfo    map[string]checks.TypeBoundInfo
}

func collectProjectSymbols(checkedPackages []checkedPkg) (map[string]*checks.GenericDefinition, map[string]bool, map[string]bool, map[string]*packageSymbols) {
	projectLocalGenerics := make(map[string]*checks.GenericDefinition)
	projectLocalGenericTypes := make(map[string]bool)
	projectImportPaths := make(map[string]bool)
	symbolsByPackage := make(map[string]*packageSymbols)

	for _, cp := range checkedPackages {
		if cp.pkg.ImportPath != "" {
			projectImportPaths[cp.pkg.ImportPath] = true
		}

		pc := &packageSymbols{
			localGenerics:     make(map[string]*checks.GenericDefinition),
			localGenericTypes: make(map[string]bool),
		}

		for _, astFile := range cp.pkg.AstFiles {
			tbi, fg, ft := checks.CollectAll(astFile)

			for k, v := range fg {
				pc.localGenerics[k] = v
			}
			for k, v := range ft {
				pc.localGenericTypes[k] = v
			}
			if pc.typeBoundsInfo == nil {
				pc.typeBoundsInfo = tbi
			} else {
				for k, v := range tbi {
					pc.typeBoundsInfo[k] = v
				}
			}

			if cp.pkg.ImportPath != "" {
				for k, v := range fg {
					projectLocalGenerics[cp.pkg.ImportPath+"."+k] = v
				}
				for k := range ft {
					projectLocalGenericTypes[cp.pkg.ImportPath+"."+k] = true
				}
			}
		}
		symbolsByPackage[cp.pkg.Dir] = pc
	}
	return projectLocalGenerics, projectLocalGenericTypes, projectImportPaths, symbolsByPackage
}

func getBasicChecks() []checks.ASTCheck {
	return []checks.ASTCheck{
		&checks.FunctionCheck{},
		&checks.MethodCheck{},
		&checks.TypeDeclCheck{},
		&checks.StructCheck{},
		&checks.TypeSetCheck{},
	}
}

func getTypeChecks() []checks.ASTCheck {
	return []checks.ASTCheck{
		checks.NewGenericFuncCallCheck(),
		checks.NewGenericTypeCompositeLitCheck(),
		&checks.GenericTypeCallCheck{},
		&checks.InstantiationDiversityCheck{},
	}
}

func groupByPackage(files []model.FileInfo) map[string][]model.FileInfo {
	groups := make(map[string][]model.FileInfo)
	for _, f := range files {
		if !strings.HasSuffix(f.Path, ".go") {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(f.Path))
		groups[dir] = append(groups[dir], f)
	}
	return groups
}

func parseFilesToASTPackage(packages map[string][]model.FileInfo, moduleName string) map[string]*model.ParsedPkg {
	dirToPkg := make(map[string]*model.ParsedPkg)
	for dir, pkgFiles := range packages {
		fset := token.NewFileSet()
		var astFiles []*ast.File
		var srcFiles []model.FileInfo
		for _, f := range pkgFiles {
			astFile, err := parser.ParseFile(fset, f.Path, f.Content, parser.AllErrors)
			if err != nil && astFile == nil {
				continue
			}
			astFiles = append(astFiles, astFile)
			srcFiles = append(srcFiles, f)
		}
		if len(astFiles) == 0 {
			continue
		}
		dirToPkg[dir] = &model.ParsedPkg{
			Dir:        dir,
			ImportPath: computeImportPath(moduleName, dir),
			Fset:       fset,
			AstFiles:   astFiles,
			SrcFiles:   srcFiles,
		}
	}
	return dirToPkg
}

func findModuleName(files []model.FileInfo) string {
	for _, f := range files {
		if filepath.Base(f.Path) != "go.mod" {
			continue
		}
		for _, line := range strings.Split(f.Content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "module "))
			}
		}
	}
	return ""
}

func computeImportPath(moduleName, dir string) string {
	if moduleName == "" {
		return ""
	}
	dir = filepath.ToSlash(dir)
	if dir == "." || dir == "" {
		return moduleName
	}
	return moduleName + "/" + dir
}

func performTypeChecking(fset *token.FileSet, files []*ast.File, importPath string, imp types.Importer) (*types.Info, *types.Package) {
	conf := types.Config{
		Importer: imp,
		Error:    func(err error) {},
	}
	info := &types.Info{
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	pkg, _ := conf.Check(importPath, fset, files, info)
	return info, pkg
}
