package firstpackage

// FirstPackage defines the generic Stack[T] type and two generic functions.
// SecondPackage imports this and instantiates them – testing cross-package
// instantiation counting (requires the import-path fix in astAnalyzer.go).
//
// Expected contributions to project-wide counters (ENABLE_TYPE_INFERENCE=true):
//   FuncTotal:                                 +2  (NewStack, Map)
//   FuncGeneric:                               +2
//   MethodTotal:                               +3  (Push, Pop, Len)
//   MethodWithGenericReceiver:                 +3
//   MethodWithGenericReceiverTrivialTypeBound: +3  (Stack[T any])
//   StructTotal:                               +1  (Stack)
//   StructGeneric:                             +1
//   TypeDecl:                                  +1  (Stack)
//   GenericTypeDecl:                           +1
//   GenericTypeInstantiationExplicit:          +2  (&Stack[T]{} in NewStack, &Stack[R]{} in Map)
//   GenericMethodInstantiationInferred:        +1  (out.Push(f(item)) in Map body)
//   – all other instantiation counters: +0

// Stack is a generic stack defined in FirstPackage.
// SecondPackage imports this and instantiates it – testing cross-package
// instantiation counting.
type Stack[T any] struct {
	items []T
}

// Push adds an item to the stack.
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop removes and returns the top item.
func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	n := len(s.items) - 1
	item := s.items[n]
	s.items = s.items[:n]
	return item, true
}

// Len returns the number of items.
func (s *Stack[T]) Len() int {
	return len(s.items)
}

// NewStack is a generic constructor function.
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

// Map applies f to every element and returns a new Stack of the results.
// It has its own type parameter R, separate from T.
func Map[T, R any](s *Stack[T], f func(T) R) *Stack[R] {
	out := &Stack[R]{}
	for _, item := range s.items {
		out.Push(f(item))
	}
	return out
}