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

	// Erweiterung 4: Generic Instantiations (nur was wirklich eckige Klammern haben kann)
	GenericFuncInstantiationExplicit         int // f[int](1) - lokale Funktionen
	GenericFuncInstantiationInferred         int // f(2) - lokale Funktionen mit Type Inference
	GenericFuncInstantiationExternalExplicit int // external.Func[int](...) - externe Funktionen
	GenericFuncInstantiationExternalInferred int // external.Func(...) - externe Funktionen mit Type Inference

	GenericTypeInstantiationExplicit         int // Box[int]{} - lokale Types
	GenericTypeInstantiationInferred         int // Box{value: 1} - lokale Types mit Type Inference
	GenericTypeInstantiationExternalExplicit int // pkg.Type[int]{} - externe Types
	GenericTypeInstantiationExternalInferred int // pkg.Type{} - externe Types mit Type Inference
}
