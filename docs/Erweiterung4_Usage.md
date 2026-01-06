# Erweiterung 4: Generic Instantiations - Usage Guide

## Überblick

Erweiterung 4 ermöglicht die Analyse von Generic Instantiations in Go-Code. Es werden zwei Arten von Instantiations gezählt:
1. **Generic Functions**: `f[int](x)` oder `f(x)`
2. **Generic Types**: `Box[int]{}` oder `Box{}`

Jeweils unterschieden nach:
- **Explizit** vs. **Inferiert** (mit/ohne eckige Klammern)
- **Lokal** vs. **Extern** (im aktuellen Package vs. importiert)

## Verwendung

### Basis-Analyse (nur explizite Instantiations)

Ohne zusätzliche Konfiguration werden nur **explizite Instantiations** gezählt:

```bash
cd GoParser
GOPARSER_SECRETS_PATH=../secret.env LOCAL_PROJECT_PATH=../LocalTestProject go run .
```

**Erkannt werden:**
- ✅ `f[int](x)` - explizite Function Instantiation
- ✅ `Box[int]{}` - explizite Type Instantiation
- ❌ `f(x)` - wird NICHT gezählt (benötigt Type Inference)
- ❌ `Box{value: 1}` - wird NICHT gezählt (benötigt Type Inference)

### Vollständige Analyse (inkl. Type Inference)

Mit aktiviertem Feature Flag werden auch **inferrierte Instantiations** gezählt:

```bash
cd GoParser
GOPARSER_SECRETS_PATH=../secret.env \
LOCAL_PROJECT_PATH=../LocalTestProject \
ENABLE_TYPE_INFERENCE=true \
go run .
```

**Erkannt werden:**
- ✅ `f[int](x)` - explizite Function Instantiation
- ✅ `Box[int]{}` - explizite Type Instantiation
- ✅ `f(x)` - inferrierte Function Instantiation
- ✅ `Box{value: 1}` - inferrierte Type Instantiation (falls Go es erlaubt)

## Feature Flag Konfiguration

### Via Environment Variable

```bash
# Aktivieren
export ENABLE_TYPE_INFERENCE=true

# Oder inline
ENABLE_TYPE_INFERENCE=true go run .
```

### Via secret.env File

Füge zu `secret.env` hinzu:

```env
ENABLE_TYPE_INFERENCE=true
```

## Output-Erklärung

### Beispiel-Output

```
Erweiterung 4 - Generic Instantiations:
  GenericFuncInstantiationExplicit: 4          # f[int](x) - lokal
  GenericFuncInstantiationInferred: 4          # f(x) - lokal
  GenericFuncInstantiationExternalExplicit: 0  # pkg.F[int](x) - extern
  GenericFuncInstantiationExternalInferred: 0  # pkg.F(x) - extern
  GenericTypeInstantiationExplicit: 2          # Box[int]{} - lokal
  GenericTypeInstantiationInferred: 0          # Box{} - lokal
  GenericTypeInstantiationExternalExplicit: 0  # pkg.Type[int]{} - extern
  GenericTypeInstantiationExternalInferred: 0  # pkg.Type{} - extern
```

### Interpretation

- **Explizit**: Entwickler schreibt die Typen manuell aus
- **Inferiert**: Go's Type Inference ermittelt die Typen automatisch
- **Lokal**: Definiert im aktuellen Package
- **Extern**: Importiert aus anderen Packages (z.B. `slices.Sort`)

## Performance-Hinweis

**Type Inference Analyse ist langsamer**, weil:
1. Vollständiges Type Checking mit `go/types` durchgeführt wird
2. Alle Abhängigkeiten aufgelöst werden müssen
3. Semantische Analyse statt nur syntaktischer Analyse

**Empfehlung**:
- Für große Projekte: `ENABLE_TYPE_INFERENCE=false` (default)
- Für detaillierte Analyse: `ENABLE_TYPE_INFERENCE=true`

## Warum keine "Generic Method Instantiations"?

In Go können **Methoden selbst keine Type Parameters haben**. Nur der Receiver-Typ kann generisch sein:

```go
// ✅ ERLAUBT: Generischer Receiver
type Box[T any] struct { value T }
func (b Box[T]) Get() T { return b.value }

// ❌ NICHT ERLAUBT: Generische Methode
type Box struct { value int }
func (b Box) Get[T any]() T { ... }  // COMPILER ERROR
```

Daher werden nur **Function** und **Type** Instantiations gezählt, keine "Method" Instantiations.

## Was wird als Type Instantiation gezählt?

Alle Stellen im Code, wo ein generischer Typ mit konkreten Type Arguments verwendet wird:

```go
type Box[T any] struct { value T }

func test() {
    // Variable Deklarationen
    var box1 Box[int]                    // ✓ GenericTypeInstantiationExplicit
    
    // Composite Literals
    box2 := Box[string]{value: "hi"}     // ✓ GenericTypeInstantiationExplicit
    
    // Type Assertions
    var x interface{} = Box[int]{value: 1}
    box3 := x.(Box[int])                 // ✓ GenericTypeInstantiationExplicit
    
    // Als Funktionsparameter/Rückgabetyp (in Deklaration, nicht in Verwendung)
    // Diese werden bereits in der Definition gezählt, nicht bei der Nutzung
}
```

## Testdatei

Eine vollständige Testdatei mit allen Cases findest du in:
`LocalTestProject/genericInstantiations.go`
