package secondpackage

import fp "localcrosspackagetestproject/FirstPackage"

func UseStackFromFirstPackage() {
	// Explicit composite literal: Stack[int]
	// GenericTypeInstantiationExplicit (cross-pkg)
	intStack := &fp.Stack[int]{}
	intStack.Push(1) 
	intStack.Push(2) 
	_, _ = intStack.Pop() 

	strStack := fp.NewStack[string]()
	strStack.Push("hello")

	_ = &fp.Stack[string]{}

	_ = fp.Map[int, string](intStack, func(n int) string {
		return string(rune(n + '0'))
	})
}