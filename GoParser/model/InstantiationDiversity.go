package model

// InstantiationDiversityRow represents one row in the instantiation diversity output:
// one generic struct or function, per repository, with its concrete and parametric usage counts.
type InstantiationDiversityRow struct {
	Repository      string
	Name            string
	Kind            string // "struct" or "func"
	ConcreteCount   int
	ParametricCount int
	ConcreteTypes   string // comma-separated list of concrete type-arg combinations
	ParametricTypes string // comma-separated list of parametric type-arg combinations
}
