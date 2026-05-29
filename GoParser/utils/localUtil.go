package utils

import (
	"GoParser/model"
	"os"
	"path/filepath"
	"strings"
)

func FetchLocalGoFiles(projectPath string) ([]model.FileInfo, error) {
	var files []model.FileInfo

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirName := info.Name()
			if dirName == "vendor" || dirName == ".git" || dirName == "node_modules" || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") || filepath.Base(path) == "go.mod" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(projectPath, path)
			if err != nil {
				relPath = path
			}
			files = append(files, model.FileInfo{
				Path:    filepath.ToSlash(relPath),
				Content: string(content),
			})
		}

		return nil
	})

	return files, err
}
