package model

import (
	"go/ast"
	"go/token"
)

type FileInfo struct {
	Path    string
	Content string
}

type ParsedPkg struct {
	Dir        string
	ImportPath string
	Fset       *token.FileSet
	AstFiles   []*ast.File
	SrcFiles   []FileInfo
}
