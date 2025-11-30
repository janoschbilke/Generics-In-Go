package main

import (
	"os"
	"path/filepath"
	"strings"
)

// fetchLocalGoFiles durchläuft ein lokales Verzeichnis rekursiv
// und sammelt alle .go-Dateien (außer vendor, .git, etc.)
func fetchLocalGoFiles(projectPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Überspringe spezielle Verzeichnisse
		if info.IsDir() {
			dirName := info.Name()
			if dirName == "vendor" || dirName == ".git" || dirName == "node_modules" || strings.HasPrefix(dirName, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Nur .go-Dateien sammeln
		if strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files = append(files, string(content))
		}

		return nil
	})

	return files, err
}
