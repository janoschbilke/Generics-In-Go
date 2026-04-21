package database

import (
	"GoParser/model"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

// InstantiationCsvDatabase writes instantiation diversity rows to a CSV file.
type InstantiationCsvDatabase struct {
	file   *os.File
	writer *csv.Writer
}

func NewInstantiationCsvDatabase(filePath string) (*InstantiationCsvDatabase, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	writer := csv.NewWriter(file)
	db := &InstantiationCsvDatabase{file: file, writer: writer}

	if err := writer.Write([]string{
		"Repository", "Name", "Kind",
		"ConcreteCount", "ParametricCount",
		"ConcreteTypes", "ParametricTypes",
	}); err != nil {
		file.Close()
		return nil, err
	}
	writer.Flush()

	return db, nil
}

// AddInstantiationData converts InstantiationData into rows and writes them sorted by name.
func (db *InstantiationCsvDatabase) AddInstantiationData(repository string, data model.InstantiationData) error {
	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, key := range names {
		entries := data[key]

		var name, kind string
		for _, e := range entries {
			name = e.Name
			kind = string(e.Kind)
			break
		}

		var concreteCombos, parametricCombos []string
		for _, entry := range entries {
			if entry.IsParametric {
				parametricCombos = append(parametricCombos, entry.TypeArgs)
			} else {
				concreteCombos = append(concreteCombos, entry.TypeArgs)
			}
		}
		sort.Strings(concreteCombos)
		sort.Strings(parametricCombos)

		record := []string{
			repository,
			name,
			kind,
			fmt.Sprintf("%d", len(concreteCombos)),
			fmt.Sprintf("%d", len(parametricCombos)),
			strings.Join(concreteCombos, "; "),
			strings.Join(parametricCombos, "; "),
		}
		if err := db.writer.Write(record); err != nil {
			return err
		}
	}
	db.writer.Flush()
	return nil
}

func (db *InstantiationCsvDatabase) Close() error {
	if db.writer != nil {
		db.writer.Flush()
	}
	if db.file != nil {
		return db.file.Close()
	}
	return nil
}
