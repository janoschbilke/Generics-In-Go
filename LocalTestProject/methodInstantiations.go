package localtestproject

type S[A any] struct{ val A }

// m is a method on S[A] – it has no own type parameters but is generic
// because its receiver type S[A] is generic.
func (x S[A]) m() A { return x.val }

// freeM is a free generic function equivalent of m, for comparison.
func freeM[A any](x S[A]) A { return x.val }

func testMethodInstantiations() {
	// S[int] instantiation via composite literal
	xInt := S[int]{val: 42}
	// Method call: receiver is S[int], so A=int is inferred
	_ = xInt.m() // GenericMethodInstantiationInferred++

	// S[string] instantiation
	xStr := S[string]{val: "hello"}
	_ = xStr.m() // GenericMethodInstantiationInferred++

	// Free generic function calls for comparison
	_ = freeM(xInt)    // GenericFuncInstantiationInferred++
	_ = freeM[int](xInt) // GenericFuncInstantiationExplicit++

	// Pointer receiver variant
	xFloat := S[float64]{val: 3.14}
	_ = xFloat.m() // GenericMethodInstantiationInferred++
}