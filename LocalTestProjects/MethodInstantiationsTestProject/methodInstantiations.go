package localtestproject

type S[A any] struct{ val A }

func (x S[A]) m() A { return x.val }

func freeM[A any](x S[A]) A { return x.val }

func testMethodInstantiations() {
	xInt := S[int]{val: 42}
	_ = xInt.m()
	xStr := S[string]{val: "hello"}
	_ = xStr.m()
	_ = freeM(xInt)
	_ = freeM[int](xInt)
	xFloat := S[float64]{val: 3.14}
	_ = xFloat.m()
}
