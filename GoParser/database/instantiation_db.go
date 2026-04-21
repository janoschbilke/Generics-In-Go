package database

import "GoParser/model"

// InstantiationDiversityDatabase stores instantiation diversity data per repository.
type InstantiationDiversityDatabase interface {
	AddInstantiationData(repository string, data model.InstantiationData) error
	Close() error
}
