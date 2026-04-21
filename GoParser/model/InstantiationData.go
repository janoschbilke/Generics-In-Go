package model

import "sort"

// InstantiationKind distinguishes between generic structs and generic functions
type InstantiationKind string

const (
	KindStruct   InstantiationKind = "struct"
	KindFunction InstantiationKind = "func"
)

// InstantiationEntry represents a single observed instantiation of a generic type or function
type InstantiationEntry struct {
	Name         string
	TypeArgs     string
	IsParametric bool
	Kind         InstantiationKind
}

func InstantiationKey(kind InstantiationKind, name string) string {
	return string(kind) + ":" + name
}

// InstantiationData maps each generic struct/function name to the set of observed instantiations.
type InstantiationData map[string]map[string]InstantiationEntry

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
