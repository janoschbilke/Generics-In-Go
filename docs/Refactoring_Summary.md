# AST Analyzer Refactoring Summary

## Ziel

Refactoring des `astAnalyzer.go` durch Einführung von Polymorphismus statt if-else/switch-case Strukturen.

## Architektur

### Check-basiertes System

Das neue System basiert auf dem **Strategy Pattern** mit folgenden Komponenten:

#### 1. Base Interface (`checks/base.go`)

```go
type ASTCheck interface {
    Check(node ast.Node, context *AnalysisContext) bool
    Update(counters *model.GenericCounters, node ast.Node, context *AnalysisContext)
}
```

#### 2. CheckRunner

Führt alle registrierten Checks auf einem AST aus:

- Iteriert über alle Nodes
- Prüft für jeden Check ob er anwendbar ist
- Führt Update aus wenn Check zutrifft

#### 3. AnalysisContext

Hält shared state für alle Checks:

- TypeBoundsInfo
- LocalGenerics
- LocalGenericTypes
- TypeInfo (für Type Inference)

### Implementierte Checks

#### Function & Method Checks (`checks/function_checks.go`)

- **FunctionCheck**: Zählt normale und generische Funktionen
- **MethodCheck**: Zählt Methoden mit generischen Receivern

#### Type Checks (`checks/type_checks.go`)

- **TypeDeclCheck**: Zählt Typ-Deklarationen
- **StructCheck**: Zählt Structs und deren Type Bounds
- **TypeSetCheck**: Erkennt TypeSets in Interfaces

#### Instantiation Checks (`checks/instantiation_checks.go`)

- **FunctionInstantiationCheck**: Erkennt generische Funktionsinstanziierungen
- **TypeInstantiationCheck**: Erkennt generische Typinstanziierungen
- **TypeInstantiationFromCallCheck**: Erkennt Type Instantiations durch Funktionsrückgabewerte

### Expression Handler System

Für die Instantiation-Checks wurde ein zusätzliches Handler-System implementiert:

#### Handler Interface (`checks/instantiation_handlers.go`)

```go
type ExpressionHandler interface {
    CanHandle(expr ast.Expr) bool
    IsExplicit() bool
    IsExternal(expr ast.Expr, context *InstantiationContext) bool
}
```

#### Implementierte Handler

- **IndexExprHandler**: `f[int]` oder `Box[int]`
- **IndexListExprHandler**: `f[int, string]`
- **InferredIdentHandler**: `f(x)` mit Type Inference
- **InferredSelectorHandler**: `pkg.Method(x)` mit Type Inference

## Vorteile der neuen Architektur

### 1. Separation of Concerns

Jeder Check hat seine eigene Klasse und Verantwortlichkeit

### 2. Open/Closed Principle

Neue Checks können hinzugefügt werden ohne bestehenden Code zu ändern

### 3. Testbarkeit

Jeder Check kann isoliert getestet werden

### 4. Lesbarkeit

- Selbstbeschreibende Namen statt verschachtelter if-else
- Klare Struktur durch Polymorphismus
- Keine Type-Switch Statements mehr

### 5. Wiederverwendbarkeit

Checks können kombiniert und wiederverwendet werden

## Migration

### Vorher

Große Switch-Statements und verschachtelte if-else Ketten im `analyzeASTAndGetCounters`

### Nachher

```go
basicChecks := []checks.ASTCheck{
    &checks.FunctionCheck{},
    &checks.MethodCheck{},
    &checks.TypeDeclCheck{},
    &checks.StructCheck{},
    &checks.TypeSetCheck{},
}

runner := checks.NewCheckRunner(basicChecks)
runner.RunChecks(file, &counters, context)
```

## Tests

Alle Tests laufen erfolgreich:

- TestASTAnalyzerBasicFunctionality
- TestASTAnalyzerWithTypeInference

## Dateien

### Neue Dateien

- `GoParser/checks/base.go` - Base Interfaces und CheckRunner
- `GoParser/checks/function_checks.go` - Function/Method Checks
- `GoParser/checks/type_checks.go` - Type/Struct/TypeSet Checks
- `GoParser/checks/instantiation_checks.go` - Instantiation Checks
- `GoParser/checks/instantiation_handlers.go` - Expression Handler
- `GoParser/checks/collectors.go` - Helper Funktionen
- `GoParser/astAnalyzer_test.go` - Unit Tests

### Geänderte Dateien

- `GoParser/astAnalyzer.go` - Komplett refactored, jetzt nur noch Orchestrierung
