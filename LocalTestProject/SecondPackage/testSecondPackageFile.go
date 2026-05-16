package secondpackage

import fp "localtestproject/FirstPackage"

// UseStackFromFirstPackage instantiates Stack[T] and calls Map[T,R] from
// FirstPackage.  These are cross-package instantiations – with the current
// single-package LocalGenericTypes they are treated as external and NOT
// counted.  This file documents the expected behaviour once cross-package
// support is added.
//
// Expected counts (after cross-package fix):
//   GenericTypeInstantiationExplicit  += 2  (Stack[int], Stack[string])
//   GenericFuncInstantiationExplicit  += 2  (NewStack[int], Map[int,string])
//   GenericMethodInstantiationInferred += 3  (Push×2, Pop×1 on Stack[int])
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