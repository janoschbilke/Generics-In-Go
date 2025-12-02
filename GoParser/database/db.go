package database

import "GoParser/model"

type genericsDatabase interface {
	AddGenericCountersEntry(repository string, data model.GenericCounters)
	Close() error
}
