package secondpackage

import fp "localcrosspackagetestproject/FirstPackage"

// UseStackFromFirstPackage instantiates Stack[T] and calls Map[T,R] from
// FirstPackage.  These are cross-package instantiations that are correctly
// counted after the import-path fix in astAnalyzer.go (conf.Check() now
// receives the full import path instead of the bare package name).
//
// Expected contributions to project-wide counters (ENABLE_TYPE_INFERENCE=true):
//   FuncTotal:                          +1  (UseStackFromFirstPackage)
//   FuncGeneric:                        +0
//   GenericTypeInstantiationExplicit:   +2  (&fp.Stack[int]{}, &fp.Stack[string]{})
//   GenericFuncInstantiationExplicit:   +2  (fp.NewStack[string](), fp.Map[int,string](...))
//   GenericMethodInstantiationInferred: +4  (intStack.Push(1), intStack.Push(2),
//                                            intStack.Pop(), strStack.Push("hello"))
//   – all other counters: +0
func UseStackFromFirstPackage() {
	// Explicit composite literal: Stack[int]
	// → GenericTypeInstantiationExplicit (cross-pkg)
	intStack := &fp.Stack[int]{}
	intStack.Push(1)  // → GenericMethodInstantiationInferred (cross-pkg, A=int)
	intStack.Push(2)  // → GenericMethodInstantiationInferred (cross-pkg, A=int)
	_, _ = intStack.Pop() // → GenericMethodInstantiationInferred (cross-pkg, A=int)

	// Explicit constructor: NewStack[string]
	// → GenericFuncInstantiationExplicit (cross-pkg)
	strStack := fp.NewStack[string]()
	strStack.Push("hello")

	// Explicit composite literal: Stack[string]
	// → GenericTypeInstantiationExplicit (cross-pkg)
	_ = &fp.Stack[string]{}

	// Explicit free-function call: Map[int, string]
	// → GenericFuncInstantiationExplicit (cross-pkg)
	_ = fp.Map[int, string](intStack, func(n int) string {
		return string(rune(n + '0'))
	})
}