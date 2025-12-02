package localtestproject

// Erweiterung 2: Struct as Type Bound

type FF struct{}

type SimpleStruct struct {
	_ int
}

// Struct mit Feldern
type ComplexStruct struct {
	_  string
	_ int
}

// CASE 1: Struct als Type Bound 
type Foo4[T FF] struct {
	_ T
}

// CASE 2: Struct mit Feldern als Type Bound
type Container[T SimpleStruct] struct {
	_ T
}

// CASE 3: Komplexere Struct als Type Bound
type Storage[T ComplexStruct] struct {
	_ []T
}