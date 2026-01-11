# Analyse Punkte

Diese Dokumentation beschreibt alle analysierten Metriken und wie sie ermittelt werden.

## Übersicht der Metriken

Der AST-Analyzer zählt folgende Metriken in Go-Projekten:

| Metrik | Beschreibung |
| -------- | -------------- |
| FuncTotal | Gesamtanzahl aller Funktionen |
| FuncGeneric | Anzahl generischer Funktionen |
| MethodTotal | Gesamtanzahl aller Methoden |
| MethodWithGenericReceiver | Anzahl Methoden mit generischem Receiver |
| MethodWithGenericReceiverTrivialTypeBound | Anzahl Methoden mit generischem Receiver und trivialem Type Bound |
| MethodWithGenericReceiverNonTrivialTypeBound | Anzahl Methoden mit generischem Receiver und non-trivial Type Bound |
| StructTotal | Gesamtanzahl aller Structs |
| StructGeneric | Anzahl generischer Structs |
| StructGenericBound | Anzahl generischer Structs mit non-trivial Type Bounds |
| StructAsTypeBound | Anzahl Structs, die als Type Bound verwendet werden |
| TypeDecl | Gesamtanzahl aller Type-Deklarationen |
| GenericTypeDecl | Anzahl generischer Type-Deklarationen |
| GenericTypeSet | Anzahl Interfaces mit Type Sets |
| **Erweiterung 4: Generic Instantiations** | |
| GenericFuncInstantiationExplicit | Explizite Instanziierungen lokaler generischer Funktionen (z.B. `f[int](x)`) |
| GenericFuncInstantiationInferred | Inferrierte Instanziierungen lokaler generischer Funktionen (z.B. `f(x)`) |
| GenericFuncInstantiationExternalExplicit | Explizite Instanziierungen externer generischer Funktionen |
| GenericFuncInstantiationExternalInferred | Inferrierte Instanziierungen externer generischer Funktionen |
| GenericTypeInstantiationExplicit | Explizite Instanziierungen lokaler generischer Types (z.B. `Box[int]{}`) |
| GenericTypeInstantiationInferred | Inferrierte Instanziierungen lokaler generischer Types (z.B. `Box{value: 1}`) |
| GenericTypeInstantiationExternalExplicit | Explizite Instanziierungen externer generischer Types |
| GenericTypeInstantiationExternalInferred | Inferrierte Instanziierungen externer generischer Types |

---

## Detaillierte Beschreibung der Metriken

### 1. Funktionen

#### FuncTotal

- **Was wird gezählt**: Alle Funktionsdeklarationen (ohne Methoden)
- **AST-Erkennung**: `*ast.FuncDecl` mit `node.Recv == nil` -> Dadurch Unterscheidung von Funktion und Methode
- **Beispiel**:

```go
func Add(a, b int) int {
    return a + b
}
```

#### FuncGeneric

- **Was wird gezählt**: Funktionen mit Type Parameters
- **AST-Erkennung**: `*ast.FuncDecl` mit `node.Type.TypeParams != nil && len(node.Type.TypeParams.List) > 0`
- **Beispiel**:

```go
func Max[T comparable](a, b T) T { 
    if a > b {
        return a
    }
    return b
}
```

---

### 2. Methoden

#### MethodTotal

- **Was wird gezählt**: Alle Methodendeklarationen
- **AST-Erkennung**: `*ast.FuncDecl` mit `node.Recv != nil`
- **Beispiel**:

```go
type MyStruct struct{}

func (m MyStruct) DoSomething() {
    // ...
}
```

#### MethodWithGenericReceiver

- **Was wird gezählt**: Methoden, deren Receiver ein generischer Typ ist
- **AST-Erkennung**: Der Receiver-Typ ist `*ast.IndexExpr` oder `*ast.IndexListExpr`
- **Beispiel**:

```go
type Container[T any] struct {
    value T
}

func (c Container[T]) Get() T { 
    return c.value
}
```

#### MethodWithGenericReceiverTrivialTypeBound (**Erweiterung 3**)

- **Was wird gezählt**: Methoden mit generischem Receiver, dessen Type Bound trivial ist
- **Trivial bedeutet**: Type Bound ist `any`, `interface{}`, oder ein leeres Interface
- **Ermittlung**:
  1. Erster Durchlauf sammelt Typinformationen über alle generischen Typen. Das ist benötigt, weil Methodenaufrufe auch vor der Typinitialisierung stehen können. Da Go keine Reihenfolge definiert, muss zuvor betrachtet werden, ob ein Struct ein non-trivial oder trivial ist
  2. Prüfung ob Type Bounds trivial sind (siehe unten)
  3. Zweiter Durchlauf klassifiziert Methoden basierend auf gesammelten Informationen
- **Beispiel**:

```go
type G[T any] struct {  // Trivialer Bound
    value T
}

func (x G[T]) someMethod() {}  // (trivial)
```

#### MethodWithGenericReceiverNonTrivialTypeBound (**Erweiterung 3**)

- **Was wird gezählt**: Methoden mit generischem Receiver, dessen Type Bound non-trivial ist
- **Non-trivial bedeutet**: Type Bound hat tatsächliche Constraints (z.B. `comparable`, Interface mit Methoden, Type Sets)
- **Ermittlung**: Analog zu trivial, aber für non-triviale Bounds
- **Beispiel**:

```go
type I[T any] interface {
    m(T)
}

type G2[T I[T]] struct {  // Non-trivialer Bound
    data T
}

func (x G2[T]) someMethod() {}  // (non-trivial)

type Comparable[T comparable] struct {  // Non-trivialer Bound
    items []T
}

func (c Comparable[T]) Contains(item T) bool {} 
```

---

### 3. Structs

#### StructTotal

- **Was wird gezählt**: Alle Struct-Deklarationen
- **AST-Erkennung**: `*ast.TypeSpec` mit `node.Type` als `*ast.StructType`
- **Beispiel**:

```go
type Person struct { 
    Name string
    Age  int
}
```

#### StructGeneric

- **Was wird gezählt**: Structs mit Type Parameters
- **AST-Erkennung**: `*ast.TypeSpec` mit Struct-Typ und `node.TypeParams != nil`
- **Beispiel**:

```go
type Container[T any] struct {  
    value T
}
```

#### StructGenericBound (**Erweiterung 1**)

- **Was wird gezählt**: Generische Structs mit mindestens einem non-trivial Type Bound
- **Non-trivial bedeutet**: Type Bound ist NICHT `any`, NICHT `interface{}`, und NICHT ein leeres Interface
- **Ermittlung**: Iteriert durch alle Type Parameters und prüft deren Constraints
- **Prüfung für triviale Constraints**:
  1. Direktes `any`: `type Foo[T any]`
  2. Direktes `interface{}`: `type Foo[T interface{}]`
  3. Leeres Interface definiert anderswo:

     ```go
     type EmptyInterface interface{}
     type Foo[T EmptyInterface] struct{}  // Gilt als trivial
     ```

- **Beispiel**:

```go
// Triviale Bounds - werden NICHT gezählt:
type Simple1[T any] struct{}
type Simple2[T interface{}] struct{}

type EmptyInterface interface{}
type Simple3[T EmptyInterface] struct{}

// Non-triviale Bounds - werden gezählt:
type Stringer interface {
    String() string
}
type Container[T Stringer] struct{}  // ✓ Wird gezählt

type Numeric[T int | float64] struct{}  // ✓ Wird gezählt

type Comparable[T comparable] struct{}  // ✓ Wird gezählt
```

#### StructAsTypeBound (**Erweiterung 2**)

- **Was wird gezählt**: Generische Structs, bei denen eine Struktur als Type Bound verwendet wird
- **Warum relevant**: Dies ist ein "unsinniger" Fall, da Structs keine Methoden in Constraints definieren können
- **Ermittlung**: Prüft ob ein Type Parameter Constraint auf eine Struct-Definition verweist
- **Beispiel**:

```go
type FF struct{} 

type Foo4[T FF] struct {  
    val T
}

type SimpleStruct struct {
    value int
}

type Container[T SimpleStruct] struct { 
    data T
}
```

---

### 4. Type-Deklarationen

#### TypeDecl

- **Was wird gezählt**: Alle Type-Deklarationen (Structs, Interfaces, Type Aliases, etc.)
- **AST-Erkennung**: `*ast.TypeSpec`
- **Beispiel**:

```go
type MyInt int
type MyInterface interface{}
type MyStruct struct{}
```

#### GenericTypeDecl

- **Was wird gezählt**: Type-Deklarationen mit Type Parameters
- **AST-Erkennung**: `*ast.TypeSpec` mit `node.TypeParams != nil`
- **Beispiel**:

```go
type Container[T any] struct { 
    value T
}

type Mapper[K comparable, V any] map[K]V 
```

---

### 5. Type Sets

#### GenericTypeSet

- **Was wird gezählt**: Interfaces, die Type Sets verwenden (Union/Intersection Types)
- **AST-Erkennung**: Interface mit `*ast.BinaryExpr` in Methods (verwendet `|` oder `&`)

- **Beispiel**:

```go
type Numeric interface {
    ~int | ~float64 
}

type SignedInteger interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 
}
```

---

## Implementierungsdetails

### Zwei-Durchlauf-Analyse (Erweiterung 3)

Für die Unterscheidung zwischen trivialen und non-trivialen Type Bounds bei Methoden verwendet der Analyzer einen zweistufigen Ansatz:

**Erster Durchlauf:**

```go
typeHasNonTrivialBound := make(map[string]bool)

// Sammelt für jeden generischen Typ:
// - Typ-Name
// - Hat er mindestens einen non-trivialen Bound?
```

**Zweiter Durchlauf:**

```go
// Analysiert Methoden und prüft:
// 1. Ist der Receiver generisch?
// 2. Welcher Typ ist der Receiver?
// 3. Hat dieser Typ triviale oder non-triviale Bounds?
```

### Trivialitätsprüfung

Ein Type Bound gilt als **trivial**, wenn:

1. Es explizit `any` ist
2. Es explizit `interface{}` ist
3. Es ein Identifier ist, der auf ein leeres Interface verweist

Ein Type Bound gilt als **non-trivial**, wenn:

1. Es ein Interface mit Methoden ist
2. Es `comparable` ist
3. Es ein Type Set ist (`int | float64`)
4. Es jede andere nicht-leere Constraint ist

### Struct als Type Bound Erkennung

Die Erkennung erfolgt durch:

1. Prüfung ob der Type Parameter Constraint ein `*ast.Ident` ist
2. Auflösung des Objekts hinter dem Identifier
3. Prüfung ob die Typ-Deklaration ein `*ast.StructType` ist

---

## Zusammenfassung der in diesem Projekt erbrachten Erweiterungen

### Erweiterung 1: StructGenericBound

**Problem**: Nicht alle generischen Structs sind gleich interessant. Structs mit trivialen Bounds (`any`, `interface{}`) sind weniger aussagekräftig. Zuvor war nur die Erkennung dieser beiden gegebenen Beispiele möglich. Ergänzt wurde, dass auch jene Fälle, in denen ein Identifier als Constraint verwendet wurde, der auf ein trivial Type zeigt (`type MyInterface interface{}`).

**Lösung**: Es wurde hinzugefügt, dass auch in solchen Fällen, der Generic Constraint als "trivial" erkannt wird.

**Impact**: Sauberere Trennnug der Fälle "trivial" und "non-trivial" Type Bounds.

---

### Erweiterung 2: StructAsTypeBound

**Problem**: Es ist möglich (aber unsinnig), eine Struct als Type Bound zu verwenden. In diesem Moment würde der Typebound schlicht als Abstraktion auf den überliegenden Type zählen.

**Lösung**: Erkenne und zähle diesen speziellen Fall separat.

**Impact**: Identifiziert potenziell problematische oder ungewöhnliche Code-Patterns.

---

### Erweiterung 3: Method Receiver Type Bound Classification

**Problem**: Nicht alle Methoden mit generischen Receivern sind gleich. Die Art der Constraints sagt viel über die Verwendung aus. Zuvor wurden Methoden lediglich als "mit generischem Receiver" klassifiziert. Es soll aber auch dort eine Möglichkeit der Unterscheidung zwischen "trivial" und "non-trivial" geben, und zwar unter Verwendung des obigen Patterns.

**Lösung**: Unterscheide zwischen:

- Methoden mit trivialem Bound (z.B. `Container[T any]`)
- Methoden mit non-trivial Bound (z.B. `Container[T comparable]`)

**Impact**:

- Besseres Verständnis der tatsächlichen Nutzung von Constraints
- Identifikation von Patterns in der Generic-Verwendung
- Erkennung von over-generic vs. properly-constrained Code

---

### Erweiterung 4: Generic Function/Method Instantiations

**Problem**: Die Verwendung von generischen Funktionen und Methoden kann auf zwei Arten erfolgen:

1. **Explizit**: `f[int](1)` - Typ ist direkt im Code sichtbar
2. **Inferiert**: `f(2)` - Typ wird durch Type Inference ermittelt

Für eine vollständige Analyse ist es wichtig zu verstehen:

- Wie oft werden Generics tatsächlich verwendet?
- Werden Type-Parameter explizit angegeben oder inferriert?
- Werden lokale oder externe generische Funktionen/Methoden verwendet?

**Lösung**: Kombinierter Ansatz mit AST + go/types:

1. **AST-basierte Erkennung** für explizite Instanziierungen (`f[int](x)`)
2. **go/types Analyse** für inferrierte Instanziierungen (`f(x)`)
3. **Unterscheidung lokal vs. extern** für bessere Einordnung

**Technische Umsetzung**:

**Phase 1 - Lokale Generics sammeln:**

```go
// Sammelt alle generischen Funktionen/Methoden im aktuellen File
localGenerics := collectLocalGenerics(file)
// Speichert: Name, isMethod flag, Anzahl Type Parameters
```

**Phase 2 - Explizite Instanziierungen (AST):**

```go
// Erkennt: f[int](x) oder obj.m[int](x)
// CallExpr.Fun ist IndexExpr oder IndexListExpr
if indexExpr, ok := callExpr.Fun.(*ast.IndexExpr); ok {
    // Explizite Typ-Angabe gefunden
}
```

**Phase 3 - Inferrierte Instanziierungen (go/types):**

```go
// Setup Type Checker
info := &types.Info{
    Instances: make(map[*ast.Ident]types.Instance),
}
// Erkennt: f(x) wo f generisch ist
if _, hasInstance := info.Instances[ident]; hasInstance {
    // Type Inference wurde verwendet
}
```

**Metriken**:

| Metrik | Beschreibung |
| -------- | -------------- |
| GenericFuncInstantiationExplicit | `f[int](x)` - lokale Funktionen |
| GenericFuncInstantiationInferred | `f(x)` - lokale Funktionen mit Type Inference |
| GenericFuncInstantiationExternalExplicit | `pkg.F[int](x)` - externe Funktionen |
| GenericFuncInstantiationExternalInferred | `pkg.F(x)` - externe Funktionen mit Type Inference |
| GenericTypeInstantiationExplicit | `Box[int]{}` - lokale Types |
| GenericTypeInstantiationInferred | `Box{value: 1}` - lokale Types mit Type Inference |
| GenericTypeInstantiationExternalExplicit | `pkg.Type[int]{}` - externe Types |
| GenericTypeInstantiationExternalInferred | `pkg.Type{}` - externe Types mit Type Inference |

**Wichtige Anmerkung zu Methoden**:
In Go können **Methoden selbst keine Type Parameters haben** - nur der Receiver-Typ kann generisch sein. Daher zählen wir keine "generic method instantiations", sondern nur generic function und generic type instantiations.

**Beispiele**:

```go
// === FUNCTION INSTANTIATIONS ===

// Generic function definition
func Print[T any](x T) {
    fmt.Printf("%v\n", x)
}

func test() {
    // Explizit - LEICHT zu erkennen (AST)
    Print[int](42)      // → GenericFuncInstantiationExplicit
    Print[string]("hi") // → GenericFuncInstantiationExplicit
    
    // Inferiert - SCHWER zu erkennen (benötigt go/types)
    Print(42)     // → GenericFuncInstantiationInferred (T = int)
    Print("hi")   // → GenericFuncInstantiationInferred (T = string)
    
    // External (z.B. slices.Sort)
    slices.Sort[int]([]int{3, 1, 2})  // → GenericFuncInstantiationExternalExplicit
    slices.Sort([]int{3, 1, 2})       // → GenericFuncInstantiationExternalInferred
}

// === TYPE INSTANTIATIONS ===

type Box[T any] struct {
    value T
}

func testTypes() {
    // Explizit - LEICHT zu erkennen (AST)
    box1 := Box[int]{value: 42}     // → GenericTypeInstantiationExplicit
    var box2 Box[string]             // → GenericTypeInstantiationExplicit
    
    // Inferiert - SCHWER zu erkennen (benötigt go/types)
    // HINWEIS: Go erlaubt Type Inference bei Composite Literals nur in bestimmten Kontexten
    // In den meisten Fällen wird explizite Typ-Angabe verlangt
    // box3 := Box{value: 1}  // Fehler: "cannot use generic type Box[T any] without instantiation"
}
```

**Feature Flag**:

- Die Analyse ist **opt-in** via `ENABLE_TYPE_INFERENCE=true`
- Grund: go/types Analyse kann langsam sein bei großen Projekten
- Default: disabled für bessere Performance

**Impact**:

- Verständnis der tatsächlichen Generic-Nutzung in der Praxis
- Identifikation von Patterns: Werden Typen explizit angegeben oder inferriert?
- Unterscheidung zwischen lokaler und externer Generic-Verwendung
- Datengrundlage für die Frage: "Lohnt sich Type Inference?"
