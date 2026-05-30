package localtestproject

import "fmt"

func GenericPrint[T any](x T) {
	fmt.Printf("%v\n", x)
}

func GenericAdd[T int | float64](a, b T) T {
	return a + b
}

type Box[T any] struct {
	value T
}

func (c Box[T]) Get() T {
	return c.value
}

func (c *Box[T]) Set(v T) {
	c.value = v
}

func test() {
	GenericPrint[int](42)
	GenericPrint[string]("hello")
	GenericPrint(42)
	GenericPrint("world")
	result1 := GenericAdd[int](1, 2)
	result2 := GenericAdd[float64](1.5, 2.5)
	result3 := GenericAdd(3, 4)
	result4 := GenericAdd(3.5, 4.5)
	box1 := Box[int]{value: 10}
	var box2 Box[string]
	box2 = Box[string]{value: "test"}
	box3 := makeBox(42)
	box4 := makeBox("hello")
	_, _, _, _, _, _, _, _ = result1, result2, result3, result4, box1, box2, box3, box4
}

func makeBox[T any](value T) Box[T] {
	return Box[T]{value: value}
}
