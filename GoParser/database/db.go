package database

import "GoParser/model"

type GenericsDatabase interface {
	AddGenericCountersEntry(data model.GenericCounters) error
	Close() error
}
