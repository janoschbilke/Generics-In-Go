package database

import (
	"GoParser/model"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	databaseObject *sql.DB
	columns        []string
}

func NewSQLiteDB(dbPath string, columns []string) (*SQLiteDB, error) {
	// Check if it is a database file
	if !strings.HasSuffix(dbPath, ".db") {
		return nil, fmt.Errorf("database file must have .db extension")
	}

	// Delete database file if it exists
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return nil, err
		}
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	sqliteDB := &SQLiteDB{databaseObject: db, columns: columns}

	if err := sqliteDB.createGenericCountersTable("generic_counters"); err != nil {
		db.Close()
		return nil, err
	}

	return sqliteDB, nil
}

func (db *SQLiteDB) createGenericCountersTable(tableName string) error {
	primaryKey := "repository"
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s STRING PRIMARY KEY, %s)",
		tableName, primaryKey, strings.Join(db.columns, ", "))

	// Append primaryKey as primary key column
	db.columns = append([]string{primaryKey}, db.columns...)

	_, err := db.databaseObject.Exec(query)
	return err
}

func (db *SQLiteDB) AddGenericCountersEntry(repository string, data model.GenericCounters) error {
	placeholders := make([]string, len(db.columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	values := make([]interface{}, len(db.columns))

	for i, col := range db.columns {
		if col == "repository" {
			values[i] = repository
			continue
		}
		v := reflect.ValueOf(data)
		t := reflect.TypeOf(data)
		colFound := false
		for j := 0; j < t.NumField(); j++ {
			jsonTag := t.Field(j).Tag.Get("json")
			if jsonTag == col {
				values[i] = v.Field(j).Interface()
				colFound = true
				break
			}
		}
		if !colFound {
			return fmt.Errorf("column %s not found in GenericCounters struct", col)
		}
	}

	query := fmt.Sprintf(
		"INSERT INTO generic_counters (%s) VALUES (%s)",
		strings.Join(db.columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := db.databaseObject.Exec(query, values...)
	return err
}

func (db *SQLiteDB) Close() error {
	if db.databaseObject != nil {
		return db.databaseObject.Close()
	}
	return nil
}
