package firstpackage

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

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

func (s *Stack[T]) Len() int {
	return len(s.items)
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func Map[T, R any](s *Stack[T], f func(T) R) *Stack[R] {
	out := &Stack[R]{}
	for _, item := range s.items {
		out.Push(f(item))
	}
	return out
}