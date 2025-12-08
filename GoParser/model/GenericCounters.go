package model

// GenericCounters represents counters for generics analysis, now with only the primary key GORM tag.
type GenericCounters struct {
	Repository  string `gorm:"primaryKey"`
	FuncTotal   int
	FuncGeneric int

	// Methoden
	MethodTotal                                  int
	MethodWithGenericReceiver                    int
	MethodWithGenericReceiverTrivialTypeBound    int // Erweiterung 3
	MethodWithGenericReceiverNonTrivialTypeBound int // Erweiterung 3

	// Structs
	StructTotal        int
	StructGeneric      int
	StructGenericBound int
	StructAsTypeBound  int // Erweiterung 2

	// Sonstiges
	TypeDecl        int
	GenericTypeDecl int
	GenericTypeSet  int
}
