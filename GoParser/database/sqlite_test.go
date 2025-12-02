package database

import (
	"GoParser/model"
	"reflect"
	"testing"
)

func TestAddGenericCountersEntries(t *testing.T) {
	var columns []string
	typeOfCounters := reflect.TypeOf(model.GenericCounters{})
	for i := 0; i < typeOfCounters.NumField(); i++ {
		columns = append(columns, typeOfCounters.Field(i).Tag.Get("json"))
	}

	db, err := NewSQLiteDB("test.db", columns)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("failed to close db: %v", err)
		}
	}()

	entry1 := model.GenericCounters{
		FuncTotal:                 123,
		FuncGeneric:               1,
		MethodTotal:               54,
		MethodWithGenericReceiver: 1,
		StructTotal:               5,
		StructGeneric:             6,
		StructGenericBound:        8,
		StructAsTypeBound:         3,
		TypeDecl:                  6,
		GenericTypeDecl:           31,
		GenericTypeSet:            3,
	}
	entry2 := model.GenericCounters{}

	if err := db.AddGenericCountersEntry("repo1", entry1); err != nil {
		t.Errorf("failed to add entry1: %v", err)
	}
	if err := db.AddGenericCountersEntry("repo2", entry2); err != nil {
		t.Errorf("failed to add entry2: %v", err)
	}
}
