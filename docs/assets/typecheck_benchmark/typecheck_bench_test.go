package typecheck_benchmark_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func generateSource(numNonGenericFunctions, numGenericFunctions, numInstantiations int) string {
	var sb strings.Builder
	sb.WriteString("package bench\n\n")
	// Generic functions
	for i := range numGenericFunctions {
		fmt.Fprintf(&sb, "func Generic%d[T any](x T) T { return x }\n", i)
	}
	sb.WriteString("\n")
	// Non-generic padding functions
	for i := range numNonGenericFunctions {
		fmt.Fprintf(&sb, "func Func%d(x int) int { return x + %d }\n", i, i)
	}
	// Call site: always calls Generic0 (only the first generic function)
	sb.WriteString("\nfunc Use() {\n")
	for i := range numInstantiations {
		fmt.Fprintf(&sb, "\t_ = Generic0(%d)\n", i)
	}
	sb.WriteString("}\n")
	return sb.String()
}

func runTypeCheck(b *testing.B, src string) {
	b.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "bench.go", src, 0)
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}
	b.ResetTimer()
	for b.Loop() {
		conf := types.Config{Error: func(error) {}}
		info := &types.Info{
			Uses:      make(map[*ast.Ident]types.Object),
			Types:     make(map[ast.Expr]types.TypeAndValue),
			Instances: make(map[*ast.Ident]types.Instance),
		}
		conf.Check("bench", fset, []*ast.File{f}, info)
	}
}

// Baseline: 1 generische Funktion, 1 Instantiierung
func BenchmarkSmallFile_OneInstantiation(b *testing.B) {
	runTypeCheck(b, generateSource(0, 1, 1))
}

// 1 generische Funktion, 1000 Instantiierungen
func BenchmarkSmallFile_ManyInstantiations(b *testing.B) {
	runTypeCheck(b, generateSource(0, 1, 1000))
}

// 1000 nicht-generische Funktionen + 1 generische, 1 Instantiierung
func BenchmarkLargeFile_OneInstantiation(b *testing.B) {
	runTypeCheck(b, generateSource(1000, 1, 1))
}

// 1000 nicht-generische Funktionen + 1 generische, 1000 Instantiierungen
func BenchmarkLargeFile_ManyInstantiations(b *testing.B) {
	runTypeCheck(b, generateSource(1000, 1, 1000))
}

// 1000 generische Funktionen, aber nur 1 wird aufgerufen (1 Instantiierung)
func BenchmarkManyGenericFunctions_OneInstantiation(b *testing.B) {
	runTypeCheck(b, generateSource(0, 1000, 1))
}

// 1000 generische Funktionen, 1000 Instantiierungen (alle rufen Generic0 auf)
func BenchmarkManyGenericFunctions_ManyInstantiations(b *testing.B) {
	runTypeCheck(b, generateSource(0, 1000, 1000))
}
