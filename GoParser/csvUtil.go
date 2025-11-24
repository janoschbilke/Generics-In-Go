package main

import (
	"encoding/csv"
	"os"
	"strings"
)

// getOwnerAndRepo liest eine CSV-Datei ein und gibt für jede Zeile owner und repo zurück
func getOwnerAndRepo(filename string) ([][2]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var result [][2]string
	for i, record := range records {
		// Überspringe die Kopfzeile
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		// Repository im Format "github.com/owner/repo"
		parts := strings.Split(record[1], "/")
		if len(parts) < 3 {
			continue
		}
		owner := parts[1]
		repo := parts[2]

		result = append(result, [2]string{owner, repo})
	}

	return result, nil
}
