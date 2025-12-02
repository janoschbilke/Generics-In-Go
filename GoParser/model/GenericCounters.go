package model

type GenericCounters struct {
	// Funktionen
	FuncTotal   int `json:"func_total"`
	FuncGeneric int `json:"func_generic"`

	// Methoden
	MethodTotal                                  int `json:"method_total"`
	MethodWithGenericReceiver                    int `json:"method_with_generic_receiver"`
	MethodWithGenericReceiverTrivialTypeBound    int `json:"method_with_generic_receiver_trivial_type_bound"`     // Erweiterung 3
	MethodWithGenericReceiverNonTrivialTypeBound int `json:"method_with_generic_receiver_non_trivial_type_bound"` // Erweiterung 3

	// Structs
	StructTotal        int `json:"struct_total"`
	StructGeneric      int `json:"struct_generic"`
	StructGenericBound int `json:"struct_generic_bound"`
	StructAsTypeBound  int `json:"struct_as_type_bound"` // Erweiterung 2

	// Sonstiges
	TypeDecl        int `json:"type_decl"`
	GenericTypeDecl int `json:"generic_type_decl"`
	GenericTypeSet  int `json:"generic_type_set"`
}
