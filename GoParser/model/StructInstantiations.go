package model

import "sort"

// InstantiationKind distinguishes between generic structs and generic functions.
type InstantiationKind string

const (
	KindStruct   InstantiationKind = "struct"
	KindFunction InstantiationKind = "func"
)

// InstantiationEntry represents a single observed instantiation of a generic type or function.
type InstantiationEntry struct {
	TypeArgs     string            // e.g. "int,string"
	IsParametric bool              // true if any type argument is itself a type parameter (e.g. T, K, V)
	Kind         InstantiationKind // "struct" or "func"
}

// InstantiationData maps each generic struct/function name to the set of observed instantiations.
// The inner map key is the type-argument combination string (e.g. "int,string").
type InstantiationData map[string]map[string]InstantiationEntry

// Merge merges src into dst (union of entries per name).
func (dst InstantiationData) Merge(src InstantiationData) {
	for name, entries := range src {
		if dst[name] == nil {
			dst[name] = make(map[string]InstantiationEntry)
		}
		for typeArgs, entry := range entries {
			dst[name][typeArgs] = entry
		}
	}
}

// ConcreteEntries returns a sorted list of concrete (non-parametric) type-arg strings for a name.
func (d InstantiationData) ConcreteEntries(name string) []string {
	var result []string
	for typeArgs, entry := range d[name] {
		if !entry.IsParametric {
			result = append(result, typeArgs)
		}
	}
	sort.Strings(result)
	return result
}

// ParametricEntries returns a sorted list of parametric type-arg strings for a name.
func (d InstantiationData) ParametricEntries(name string) []string {
	var result []string
	for typeArgs, entry := range d[name] {
		if entry.IsParametric {
			result = append(result, typeArgs)
		}
	}
	sort.Strings(result)
	return result
}
