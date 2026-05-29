package importer

import (
	"fmt"
	"go/types"
	"sync"
)


type ProjectImporter struct {
	mu       sync.Mutex
	packages map[string]*types.Package
}

func New() *ProjectImporter {
	return &ProjectImporter{packages: make(map[string]*types.Package)}
}

func (i *ProjectImporter) Import(path string) (*types.Package, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if pkg, ok := i.packages[path]; ok {
		return pkg, nil
	}
	return nil, fmt.Errorf("package %q not found in project", path)
}

func (i *ProjectImporter) AddPackage(importPath string, pkg *types.Package) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.packages[importPath] = pkg
}
