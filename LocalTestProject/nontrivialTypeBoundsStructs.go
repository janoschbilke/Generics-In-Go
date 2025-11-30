package localtestproject

// NON-TRIVIAL BOUND: Interface mit tatsächlichen Methoden
type Stringer interface {
	String() string
}

type StringableContainer[T Stringer] struct {
	value T
}

// NON-TRIVIAL BOUND: Interface mit Type Set
type NumericInterface interface {
	~int | ~float64
}

type NumericContainer[T NumericInterface] struct {
	number T
}

// Methods with Generic Receiver

func (s *StringableContainer[T]) GetString() string {
	return s.value.String()
}

func (n *NumericContainer[T]) Double() T {
	return n.number + n.number
}
