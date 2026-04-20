package utils

import (
	"GoParser/database"
	"GoParser/model"
	"reflect"
)

func CreateDatabase(config SetupConfiguration) (database.GenericsDatabase, error) {
	var genericsDatabase database.GenericsDatabase
	var err error

	if config.OutputFormat == "csv" {
		genericsDatabase, err = database.NewCsvDatabase("generic_counters.csv")
	} else {
		genericsDatabase, err = database.NewSQLiteDB("generic_counters.db")
	}

	if err != nil {
		return nil, err
	}

	return genericsDatabase, nil
}

func GetColumns() []string {
	var columns []string
	typeOfCounters := reflect.TypeOf(model.GenericCounters{})
	for i := 0; i < typeOfCounters.NumField(); i++ {
		columns = append(columns, typeOfCounters.Field(i).Tag.Get("json"))
	}
	return columns
}
