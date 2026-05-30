package localtestproject

type FF struct{}

type SimpleStruct struct {
	_ int
}

type ComplexStruct struct {
	_ string
	_ int
}

type Foo4[T FF] struct {
	_ T
}

type Container[T SimpleStruct] struct {
	_ T
}

type Storage[T ComplexStruct] struct {
	_ []T
}
