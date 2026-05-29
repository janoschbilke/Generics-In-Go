package utils

import (
	"GoParser/model"
	"strings"
)

func TopoSortPackages(dirToPkg map[string]*model.ParsedPkg, _ string) []string {
	importToDir := make(map[string]string)
	for dir, pkg := range dirToPkg {
		if pkg.ImportPath != "" {
			importToDir[pkg.ImportPath] = dir
		}
	}

	deps := make(map[string][]string)
	for dir, pkg := range dirToPkg {
		for _, astFile := range pkg.AstFiles {
			for _, imp := range astFile.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if depDir, ok := importToDir[path]; ok && depDir != dir {
					deps[dir] = append(deps[dir], depDir)
				}
			}
		}
	}

	inDegree2 := make(map[string]int)
	for dir := range dirToPkg {
		inDegree2[dir] = 0
	}
	revDeps := make(map[string][]string)
	for dir, depDirs := range deps {
		for _, d := range depDirs {
			revDeps[d] = append(revDeps[d], dir)
			inDegree2[dir]++
		}
	}

	var queue []string
	for dir := range dirToPkg {
		if inDegree2[dir] == 0 {
			queue = append(queue, dir)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		sorted = append(sorted, cur)
		for _, dependent := range revDeps[cur] {
			inDegree2[dependent]--
			if inDegree2[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	inSorted := make(map[string]bool)
	for _, d := range sorted {
		inSorted[d] = true
	}
	for dir := range dirToPkg {
		if !inSorted[dir] {
			sorted = append(sorted, dir)
		}
	}

	return sorted
}
