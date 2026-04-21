package utils

import (
	"GoParser/database"
)

// DatabaseSet holds all database handles created for a single run
type DatabaseSet struct {
	Generics       database.GenericsDatabase
	Instantiations database.InstantiationDiversityDatabase
}

func (d *DatabaseSet) Close() error {
	if d.Generics != nil {
		if err := d.Generics.Close(); err != nil {
			return err
		}
	}
	if d.Instantiations != nil {
		if err := d.Instantiations.Close(); err != nil {
			return err
		}
	}
	return nil
}

func CreateDatabases(config SetupConfiguration) (*DatabaseSet, error) {
	set := &DatabaseSet{}
	var err error

	if config.OutputFormat == "csv" {
		set.Generics, err = database.NewCsvDatabase("generic_counters.csv")
	} else {
		set.Generics, err = database.NewSQLiteDB("generic_counters.db")
	}
	if err != nil {
		return nil, err
	}

	if config.EnableTypeInference && config.OutputFormat == "csv" {
		set.Instantiations, err = database.NewInstantiationCsvDatabase("generic_counters_instantiation.csv")
		if err != nil {
			_ = set.Generics.Close()
			return nil, err
		}
	} else if config.EnableTypeInference {
		// TODO: SQLite — instantiation diversity as a new table in the existing DB not yet implemented
	}

	return set, nil
}
