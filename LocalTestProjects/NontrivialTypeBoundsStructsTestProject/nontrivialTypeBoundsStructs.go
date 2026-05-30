package localtestproject

type Stringer interface {
	String() string
}

type StringableContainer[T Stringer] struct {
	value T
}

type NumericInterface interface {
	~int | ~float64
}

type NumericContainer[T NumericInterface] struct {
	number T
}

func (s *StringableContainer[T]) GetString() string {
	return s.value.String()
}

func (n *NumericContainer[T]) Double() T {
	return n.number + n.number
}