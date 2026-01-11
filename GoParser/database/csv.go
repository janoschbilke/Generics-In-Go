package database

import (
	"GoParser/model"
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
)

type CsvDatabase struct {
	file   *os.File
	writer *csv.Writer
}

func NewCsvDatabase(filePath string) (*CsvDatabase, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	writer := csv.NewWriter(file)
	db := &CsvDatabase{file: file, writer: writer}
	if err := db.printHeader(model.GenericCounters{}); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *CsvDatabase) printHeader(data interface{}) error {
	v := reflect.TypeOf(data)
	n := v.NumField()
	header := make([]string, n)
	for i := 0; i < n; i++ {
		header[i] = v.Field(i).Name
	}
	if err := db.writer.Write(header); err != nil {
		return err
	}
	db.writer.Flush()
	return nil
}

func (db *CsvDatabase) AddGenericCountersEntry(data model.GenericCounters) error {
	v := reflect.ValueOf(data)
	n := v.NumField()
	csvRow := make([]string, n)
	for i := 0; i < n; i++ {
		field := v.Field(i)
		csvRow[i] = fmt.Sprintf("%v", field.Interface())
	}
	if err := db.writer.Write(csvRow); err != nil {
		return err
	}
	db.writer.Flush()
	return nil
}

func (db *CsvDatabase) Close() error {
	if db.writer != nil {
		db.writer.Flush()
	}
	if db.file != nil {
		return db.file.Close()
	}
	return nil
}
