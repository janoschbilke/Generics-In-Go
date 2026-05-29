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

	// Build project-wide maps (qualified: "<import/path>.<Name>") so that
	// cross-package instantiations can be recognised by the handlers.
	projectLocalGenerics := make(map[string]*checks.GenericDefinition)
	projectLocalGenericTypes := make(map[string]bool)
	projectImportPaths := make(map[string]bool)
	for _, cp := range checkedPackages {
		if cp.pkg.ImportPath != "" {
			projectImportPaths[cp.pkg.ImportPath] = true
		}
		for _, astFile := range cp.pkg.AstFiles {
			_, fg, ft := checks.CollectAll(astFile)
			for k, v := range fg {
				projectLocalGenerics[cp.pkg.ImportPath+"."+k] = v
			}
			for k := range ft {
				projectLocalGenericTypes[cp.pkg.ImportPath+"."+k] = true
			}
		}
	}

	for _, cp := range checkedPackages {
		pkgLocalGenerics := make(map[string]*checks.GenericDefinition)
		pkgLocalGenericTypes := make(map[string]bool)
		var pkgTypeBoundsInfo map[string]checks.TypeBoundInfo

		for _, astFile := range cp.pkg.AstFiles {
			tbi, fg, ft := checks.CollectAll(astFile)
			for k, v := range fg {
				pkgLocalGenerics[k] = v
			}
			for k, v := range ft {
				pkgLocalGenericTypes[k] = v
			}
			if pkgTypeBoundsInfo == nil {
				pkgTypeBoundsInfo = tbi
			} else {
				for k, v := range tbi {
					pkgTypeBoundsInfo[k] = v
				}
			}
		}

		allChecks := []checks.ASTCheck{}
		allChecks = append(allChecks, getBasicChecks()...)
		allChecks = append(allChecks, getTypeChecks()...)
		runner := checks.NewCheckRunner(allChecks)

		for _, astFile := range cp.pkg.AstFiles {
			ctx := &checks.AnalysisContext{
				TypeBoundsInfo:           pkgTypeBoundsInfo,
				LocalGenerics:            pkgLocalGenerics,
				LocalGenericTypes:        pkgLocalGenericTypes,
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
		pkgName := pkg.AstFiles[0].Name.Name
		typeInfo, typedPkg := performTypeChecking(pkg.Fset, pkg.AstFiles, pkgName, projectImporter)
		if typedPkg != nil && pkg.ImportPath != "" {
			projectImporter.AddPackage(pkg.ImportPath, typedPkg)
		}
		checked = append(checked, checkedPkg{pkg: pkg, typeInfo: typeInfo})
	}
	return checked
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

func performTypeChecking(fset *token.FileSet, files []*ast.File, pkgName string, imp types.Importer) (*types.Info, *types.Package) {
	conf := types.Config{
		Importer: imp,
		Error:    func(err error) {},
	}
	info := &types.Info{
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	pkg, _ := conf.Check(pkgName, fset, files, info)
	return info, pkg
}
