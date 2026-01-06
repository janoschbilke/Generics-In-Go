package localtestproject

import (
	"fmt"
	"slices"
	"sort"
)

func testExternalGenerics() {
	// === EXTERNAL FUNCTION INSTANTIATIONS ===
	
	numbers := []int{3, 1, 4, 1, 5, 9, 2, 6}
	
	slices.Sort[[]int, int](numbers)
	slices.Reverse[[]int](numbers)
	
	slices.Sort(numbers)
	slices.Reverse(numbers)
	max := slices.Max(numbers)
	min := slices.Min(numbers)
	
	strings := []string{"apple", "banana", "cherry"}
	slices.Sort(strings)
	contains := slices.Contains(strings, "banana")
	
	index := slices.Index[[]string, string](strings, "cherry")
	
	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i] < numbers[j]
	})
	
	m := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}
	
	fmt.Printf("Max: %d, Min: %d, Contains: %v, Index: %d\n", max, min, contains, index)
	
	_ = numbers
	_ = strings
	_ = m
}

func testSlicesClone() {
	original := []int{1, 2, 3, 4, 5}
	
	cloned1 := slices.Clone[[]int](original)
	cloned2 := slices.Clone(original)
	
	fmt.Printf("Original: %v, Cloned1: %v, Cloned2: %v\n", original, cloned1, cloned2)
}

func testSlicesCompact() {
	data := []int{1, 1, 2, 2, 2, 3, 1, 1}
	
	compacted := slices.Compact(data)
	
	fmt.Printf("Original: %v, Compacted: %v\n", data, compacted)
}

func testSlicesEqual() {
	slice1 := []int{1, 2, 3}
	slice2 := []int{1, 2, 3}
	slice3 := []int{1, 2, 4}
	
	equal1 := slices.Equal(slice1, slice2)
	equal2 := slices.Equal(slice1, slice3)
	
	fmt.Printf("Equal1: %v, Equal2: %v\n", equal1, equal2)
}

type Person struct {
	Name string
	Age  int
}

func testCustomTypeWithExternalGenerics() {
	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 35},
	}
	
	slices.SortFunc(people, func(a, b Person) int {
		return a.Age - b.Age
	})
	
	hasYoung := slices.ContainsFunc(people, func(p Person) bool {
		return p.Age < 30
	})
	
	fmt.Printf("Sorted people: %v, Has young: %v\n", people, hasYoung)
}
