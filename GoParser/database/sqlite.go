package database

import (
	"GoParser/model"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDB struct {
	databaseObject *gorm.DB
}

func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
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
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the GenericCounters struct
	if err := db.AutoMigrate(&model.GenericCounters{}); err != nil {
		return nil, err
	}

	return &SQLiteDB{databaseObject: db}, nil
}

func (db *SQLiteDB) AddGenericCountersEntry(data model.GenericCounters) error {
	return db.databaseObject.Create(&data).Error
}

func (db *SQLiteDB) Close() error {
	// GORM does not have a direct Close method, but you can get the underlying sql.DB
	if db.databaseObject != nil {
		sqlDB, err := db.databaseObject.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
