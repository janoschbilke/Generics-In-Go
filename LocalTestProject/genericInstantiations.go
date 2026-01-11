package localtestproject

import "fmt"

// Erweiterung 4: Generic Function and Method Instantiations

// Generic function definition
func GenericPrint[T any](x T) {
	fmt.Printf("%v\n", x)
}

// Generic function with constraint
func GenericAdd[T int | float64](a, b T) T {
	return a + b
}

// Generic struct with method
type Box[T any] struct {
	value T
}

func (c Box[T]) Get() T {
	return c.value
}

func (c *Box[T]) Set(v T) {
	c.value = v
}

// Test function to trigger instantiations
func test() {
	// === FUNCTION INSTANTIATIONS ===
	
	// Explicit 
	GenericPrint[int](42) 
	GenericPrint[string]("hello") 
	
	// Inferred 
	GenericPrint(42)
	GenericPrint("world")
	
	// More explicit
	result1 := GenericAdd[int](1, 2) 
	result2 := GenericAdd[float64](1.5, 2.5) 
	
	// More inferred
	result3 := GenericAdd(3, 4)       
	result4 := GenericAdd(3.5, 4.5)   

	// === TYPE INSTANTIATIONS ===
	
	// Explicit 
	box1 := Box[int]{value: 10} 
	
	// Variable declaration with explicit type
	var box2 Box[string]  
	box2 = Box[string]{value: "test"}  
	
	// Inferred 
	box3 := makeBox(42)  
	box4 := makeBox("hello") 
	
	_, _, _, _, _, _, _, _ = result1, result2, result3, result4, box1, box2, box3, box4
}

func makeBox[T any](value T) Box[T] {
	return Box[T]{value: value}
}
