# Generic Analysis Tool

## 1. Analyse-Walkthrough anhand eines konkreten Beispiels

### 1.1 Das Beispiel

Gegeben sei ein kleines Go-Projekt mit zwei Dateien in zwei Paketen:

```go
// collections/stack.go
package collections

type Stack[T any] struct { // -> Generisches Struct
    items []T
}

func (s *Stack[T]) Push(item T) { // -> Generische Methode
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T { // -> Generische Methode
    item := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return item
}
```

```go
// main.go
package main

import "myproject/collections"

func main() {
    s := collections.Stack[int]{}  // explizite Typ-Instanziierung
    s.Push(42)                     // inferierte Methoden-Instanziierung
    val := s.Pop()                 // inferierte Methoden-Instanziierung
    _ = val
}
```

---

### 1.2 Schritt 1: `groupByPackage`

Die Quelldateien werden nach Verzeichnis gruppiert:

```text
map[dir][]FileInfo:
  "collections" -> [stack.go]
  "."           -> [main.go]
```

---

### 1.3 Schritt 2: `parseFilesToASTPackage`

Für jedes Verzeichnis wird `parser.ParseFile` aufgerufen. Der resultierende AST für `stack.go` sieht strukturell so aus:

```text
*ast.File (stack.go)
└── Decls
    ├── *ast.GenDecl (TypeSpec)
    │   └── TypeSpec{Name: "Stack", TypeParams: [T any]}
    │       └── StructType{Fields: [items []T]}
    │
    ├── *ast.FuncDecl (Push)
    │   ├── Recv: *Stack[T]
    │   └── Body: ...
    │
    └── *ast.FuncDecl (Pop)
        ├── Recv: *Stack[T]
        └── Body: ...
```

Wichtig: Der AST enthält die Typ-Parameter `[T any]` syntaktisch, aber **keine Information darüber, mit welchen konkreten Typen `Stack` instanziiert wird**. Das ist im AST von `main.go` nicht direkt auflösbar.

---

### 1.4 Schritt 3: `TopoSortPackages`

Die Pakete werden topologisch sortiert. `main` importiert `collections`, also:

```text
Reihenfolge: ["collections", "."]
```

`collections` wird zuerst gecheckt, damit `main` es als Dependency auflösen kann.

---

### 1.5 Schritt 4: `extractTypeInfoForPackages` – N × `types.Check`

Für jedes Paket in topologischer Reihenfolge wird `types.Config.Check()` aufgerufen.

**Aufruf 1: `types.Check` für Paket `collections`**

```text
Input:  [stack.go AST]
Output: types.Info{
    Uses:      { Ident("T") -> TypeParam T, Ident("append") -> builtin, ... }
    Types:     { Expr("s.items") -> []T, Expr("item") -> T, ... }
    Instances: {}   // keine Instantiierungen in der Definition selbst
}
typedPkg -> wird in projectImporter gespeichert
```

**Aufruf 2: `types.Check` für Paket `main`**

```text
Input:  [main.go AST], Importer kennt bereits "collections"
Output: types.Info{
    Uses:      { Ident("Stack") -> TypeName collections.Stack, Ident("s") -> Var s, ... }
    Types:     { Expr("s") -> *collections.Stack[int], Expr("42") -> untyped int, ... }
    Instances: {
        Ident("Stack") -> Instance{ TypeArgs: [int] }
        Ident("Push")  -> Instance{ TypeArgs: [int] }
        Ident("Pop")   -> Instance{ TypeArgs: [int] }
    }
}
```

`Uses` und `Types` werden für alle Bezeichner und Ausdrücke im gesamten File befüllt.Alle drei Maps werden aktiv genutzt:

- `Instances`: primär für die Instantiierungserkennung: welche Typ-Argumente wurden für einen generischen Bezeichner inferiert?
- `Uses`: um aufzulösen ob ein Bezeichner auf ein `*types.PkgName` zeigt (Cross-Package-Referenzen) und ob ein Methoden-Receiver ein bekannter generischer Typ ist
- `Types`: um den konkreten Typ eines Ausdrucks aufzulösen, z.B. dass `s` in `s.Push(42)` vom Typ `Stack[int]` ist; nötig für die Erkennung von Methoden-Instantiierungen auf generischen Receivern

Genau hier liegt der Kern: `types.Check` inferiert, dass `s.Push(42)` eine Instanziierung von `Push[int]` ist, obwohl `[int]` im Quellcode nicht steht. Das ist nur möglich, weil `s` als `Stack[int]` bekannt ist, und das wiederum nur, weil `collections` bereits gecheckt und im `projectImporter` verfügbar ist.

---

### 1.6 Schritt 5: `collectProjectSymbols`

Für alle gecheckte Pakete wird `CollectAll` aufgerufen (ein einziger `ast.Inspect`-Walk pro Datei), der drei Maps gleichzeitig befüllt:

```text
projectLocalGenerics:
  "myproject/collections.Stack" -> GenericDefinition{TypeParams: [T]}
  "myproject/collections.Push"  -> GenericDefinition{TypeParams: [T]}
  "myproject/collections.Pop"   -> GenericDefinition{TypeParams: [T]}

projectLocalGenericTypes:
  "myproject/collections.Stack" -> true

projectImportPaths:
  "myproject/collections" -> true
  "myproject"             -> true
```

Diese Maps ermöglichen es den Checks, Cross-Package-Instantiierungen zu erkennen: Wenn in `main.go` `Stack[int]{}` steht, kann der Check nachschlagen, dass `Stack` aus einem bekannten Paket stammt und generisch ist.

---

### 1.7 Schritt 6: CheckRunner

Für jede Datei jedes Pakets werden alle Checks ausgeführt. Die relevanten Ergebnisse:

**`TypeDeclCheck` auf `stack.go`:**

- Findet `TypeSpec` mit `TypeParams != nil` → `GenericTypeDecl++`

**`MethodCheck` auf `stack.go`:**

- Findet `Push` mit Receiver `*Stack[T]` → `MethodWithGenericReceiver++`
- Findet `Pop` mit Receiver `*Stack[T]` → `MethodWithGenericReceiver++`

**`GenericTypeCompositeLitCheck` auf `main.go`:**

- Findet `Stack[int]{}` → `GenericTypeInstantiationExplicit++`

**`GenericFuncCallCheck` auf `main.go`:**

- Findet `s.Push(42)`: schlägt in `types.Info.Instances` nach → `Instance{TypeArgs: [int]}` → `GenericFuncInstantiationInferred++`
- Findet `s.Pop()`: schlägt in `types.Info.Instances` nach → `Instance{TypeArgs: [int]}` → `GenericFuncInstantiationInferred++`

**`InstantiationDiversityCheck` auf `main.go`:**

- Registriert: `Stack` wurde mit `[int]` instanziiert

**Endergebnis für dieses Projekt:**

```text
GenericTypeDecl:                  1   (Stack[T])
MethodWithGenericReceiver:        2   (Push, Pop)
GenericTypeInstantiationExplicit: 1   (Stack[int]{})
GenericFuncInstantiationInferred: 2   (Push(42), Pop())
```

---

### 1.8 Warum implizite Instanziierungen nicht direkt aus dem AST lesbar sind

```go
s.Push(42)   // Im AST: CallExpr{ Fun: SelectorExpr{ X: "s", Sel: "Push" }, Args: [42] }
```

Der AST-Node für `Push` ist ein `*ast.Ident` mit dem Namen `"Push"`. Er enthält **keine** Information darüber, dass `Push` generisch ist oder dass `T=int`. Ohne `types.Info.Instances` ist dieser Aufruf von einem nicht-generischen Methodenaufruf nicht zu unterscheiden.

```go
s := Stack[int]{}   // Im AST sichtbar: IndexExpr{ X: "Stack", Index: "int" }
s.Push(42)          // Im AST NICHT sichtbar: kein [int] im Aufruf
```

---

## 2. Timing-Analyse

### 2.1 Gemessene Phasen

Für jedes analysierte Projekt werden folgende Phasen gemessen und in `output/timing.csv` gespeichert:

| Phase           | Feld                 | Beschreibung                                     |
| --------------- | -------------------- | ------------------------------------------------ |
| Fetch Files     | `fetch_files_ms`     | GitHub API-Aufrufe oder lokales FS-Lesen         |
| Parse AST       | `parse_ast_ms`       | `parser.ParseFile` für alle Dateien aller Pakete |
| Topo Sort       | `topo_sort_ms`       | Topologische Sortierung der Paket-Abhängigkeiten |
| Type Check      | `type_check_ms`      | N_packages × `types.Config.Check()`              |
| Collect Symbols | `collect_symbols_ms` | Aufbau der projekt-weiten Generic-Maps           |
| Check Runner    | `check_runner_ms`    | Alle AST-Checks über alle Dateien                |
| Total Analysis  | `total_analysis_ms`  | Summe aller Analyse-Phasen (ohne Fetch)          |
| Total           | `total_ms`           | Fetch + Total Analysis                           |

### 2.2 Ergebnisse (Auswahl aus 100 analysierten Repos)

| Repository            | fetch_ms | parse_ast_ms | type_check_ms | check_runner_ms | total_analysis_ms | total_ms | Gen. Def.\* | Inst.\*\* |
| --------------------- | -------- | ------------ | ------------- | --------------- | ----------------- | -------- | ----------- | --------- |
| kubernetes/kubernetes | 8729     | 1324         | 1951          | 478             | 3895              | 12625    | 818         | 89.870    |
| ollama/ollama         | 3209     | 116          | 192           | 40              | 361               | 3571     | 57          | 9.241     |
| jesseduffield/lazygit | 1525     | 53           | 115           | 22              | 196               | 1721     | 114         | 28.012    |
| gohugoio/hugo         | 2312     | 77           | 111           | 28              | 225               | 2538     | 301         | 9.010     |
| syncthing/syncthing   | 1437     | 57           | 83            | 18              | 164               | 1602     | 33          | 7.385     |
| netdata/netdata       | 3807     | 202          | 308           | 68              | 600               | 4408     | 155         | 620       |
| junegunn/fzf          | 667      | 25           | 35            | 6               | 69                | 736      | 7           | 1.921     |
| anthdm/hbbft          | 725      | 4            | 5             | 0               | 11                | 736      | 0           | 195       |
| joesonw/go-diff       | 640      | 0            | 0             | 0               | 0                 | 640      | 0           | 0         |

\*Gen. Def. = FuncGeneric + MethodWithGenericReceiver + StructGeneric + GenericTypeDecl
\*\*Inst. = GenericFuncInstantiationExplicit + GenericFuncInstantiationInferred + GenericTypeInstantiationExplicit + GenericTypeInstantiationInferred + GenericMethodInstantiationInferred

### 2.3 Interpretation

**Fetch dominiert die Gesamtlaufzeit.**
Im GitHub-Modus macht `fetch_files_ms` bei den meisten Repos 70–95 % der Gesamtlaufzeit aus. Für `kubernetes/kubernetes` sind es 8,7 s von 12,6 s Gesamtzeit. Die eigentliche Analyse (3,9 s) ist schneller als das Laden der Dateien. Das liegt daran, dass für jede Datei ein separater GitHub-API-Aufruf nötig ist.

**`type_check_ms` ist die teuerste Analyse-Phase.**
Innerhalb von `total_analysis_ms` entfallen typischerweise ~50 % auf `type_check_ms`, ~30 % auf `parse_ast_ms` und ~20 % auf `check_runner_ms`. `topo_sort_ms` und `collect_symbols_ms` sind in allen Fällen vernachlässigbar (< 5 % von `total_analysis_ms`).

**Repos ohne Generics: Analyse < 1 ms.**
Projekte wie `joesonw/go-diff` (3 Funktionen, keine Generics) zeigen 0 ms für alle Analyse-Phasen. Das ist kein Messfehler, sondern ein Auflösungsproblem der Millisekunden-Granularität, die Analyse läuft in unter 1 ms durch.

**Korrelation zwischen Projektgröße und `type_check_ms`.**
`kubernetes/kubernetes` hat 50.884 Funktionen und braucht 1.951 ms für `type_check`. `junegunn/fzf` hat 403 Funktionen und braucht 35 ms. Das Verhältnis (~56×) entspricht grob dem Verhältnis der Funktionsanzahl (~126×), was zeigt, dass `types.Check` annähernd linear mit der Paketgröße skaliert.

---

## 3. Wie `types.Check` funktioniert

### 3.1 Was `types.Check` analysiert

`types.Config.Check(importPath, fset, files, info)` führt für die übergebenen Dateien einen vollständigen semantischen Analyse-Pass durch:

1. **Name Resolution** – Alle Bezeichner werden ihren Deklarationen zugeordnet [2]
2. **Type Inference** – Typ-Parameter werden aus dem Kontext inferiert [3]
3. **Constraint Checking** – Typ-Argumente werden gegen ihre Constraints geprüft [4]
4. **Instantiation Recording** – Alle generischen Instantiierungen werden in `info.Instances` eingetragen [2]

### 3.2 Analysiert `Check` immer das gesamte Paket?

**Ja.** `types.Check` analysiert immer alle übergebenen Dateien als Einheit. Es gibt keine Möglichkeit, nur einen einzelnen AST-Knoten zu checken. Selbst wenn man nur an einer einzigen Instantiierung interessiert ist, muss das gesamte Paket gecheckt werden.

```text
Naive Implementierung (alter Ansatz):
    for each file:
        types.Check([file])     // N_files Aufrufe, jede Datei isoliert

Optimierte Implementierung (neuer Ansatz):
    for each package (in topo. Reihenfolge):
        types.Check(package.allFiles)   // N_packages Aufrufe, ganzes Paket
```

Der Unterschied: Bei einem Paket mit 50 Dateien reduziert sich die Anzahl der `types.Check`-Aufrufe von 50 auf 1. Außerdem kann der Type-Checker im neuen Ansatz Typen über Dateigrenzen hinweg auflösen.

**Formale Betrachtung als Mengen:**

Sei P ein Go-Paket mit Dateien F = {f_1, ..., f_m}. Jede Datei f_i wird zu einem AST mit einer Menge von Knoten N_i geparst. Die Gesamtmenge aller Knoten, die `types.Check` übergeben wird:

N = N_1 ∪ N_2 ∪ ... ∪ N_m = {n_1, ..., n_k}

Innerhalb von N unterscheiden wir:

- **G ⊆ N** – generische Definitionen (Funktionen/Typen mit Typ-Parametern)
- **I ⊆ N** – Instantiierungsstellen (Call-Expressions, Composite Literals)
- **R = N \ (G ∪ I)** – alle übrigen nicht-generischen Knoten

Die naive Annahme wäre, dass `types.Check` nur G ∪ I verarbeiten muss. Die Realität: `types.Check` verarbeitet immer die gesamte Menge N, unabhängig von |G| und |I|. Der Benchmark in Section 3.3 bestätigt dies empirisch: Laufzeit ∝ |N|, nicht ∝ |G| oder |I|.

### 3.3 Benchmark: Skaliert `Check` mit Dateigröße oder Instantiierungsanzahl?

Um zu verstehen, womit `types.Check` skaliert, wurde ein Mikrobenchmark mit einer 2×2-Matrix durchgeführt (Quellcode: `docs/assets/typecheck_benchmark/`). Synthetisch generierte Go-Dateien variieren unabhängig voneinander in Dateigröße (Anzahl nicht-generischer Funktionen) und Anzahl generischer Instantiierungen.

Das `types.Info`-Struct im Benchmark entspricht dem echten Analyzer (`Uses`, `Types`, `Instances` alle non-nil). Die offizielle Dokumentation erklärt warum das relevant ist:

> *"Check populates each of the non-nil maps in the Info struct."* [1]

Jede non-nil Map bedeutet zusätzliche Arbeit für `Check`. Der Benchmark misst daher die reale Laufzeit des Analyzers, nicht eine vereinfachte Variante.

**Ergebnisse** (Apple M4 Pro, Go 1.22, `go test -bench=. -benchtime=3s`):

| Szenario                                | Nicht-generische Fns | Generische Fns | Instantiierungen | ns/op     | Faktor   |
| --------------------------------------- | -------------------- | -------------- | ---------------- | --------- | -------- |
| Baseline                                | 0                    | 1              | 1                | 5.376     | 1×       |
| Viele Instantiierungen                  | 0                    | 1              | 1.000            | 2.447.963 | **455×** |
| Viele nicht-generische Fns, 1 Inst.     | 1.000                | 1              | 1                | 2.175.650 | **405×** |
| Viele nicht-generische Fns, viele Inst. | 1.000                | 1              | 1.000            | 5.054.883 | **940×** |
| Viele generische Fns, 1 Inst.           | 0                    | 1.000          | 1                | 2.134.096 | **397×** |
| Viele generische Fns, viele Inst.       | 0                    | 1.000          | 1.000            | 4.635.733 | **862×** |

**Interpretation:**

`types.Check` skaliert mit **allen drei** Dimensionen. Die entscheidende Erkenntnis liefert der Vergleich der letzten vier Zeilen:

- 1.000 nicht-generische Fns + 1 Inst. → **2.175.650 ns**
- 1.000 generische Fns + 1 Inst. → **2.134.096 ns**

Diese beiden Werte sind nahezu identisch (~2% Unterschied). `types.Check` unterscheidet **nicht** zwischen generischen und nicht-generischen Funktionen – es verarbeitet alle Deklarationen im File, unabhängig davon ob sie generisch sind oder ob sie überhaupt aufgerufen werden. Die 999 nicht-aufgerufenen generischen Funktionen werden trotzdem vollständig gecheckt.

Es gibt keine "intelligente" Abkürzung: Selbst wenn nur eine einzige generische Funktion im gesamten File instantiiert wird, muss `Check` alle anderen Funktionen ebenfalls auflösen, typisieren und in die `Uses`- und `Types`-Maps eintragen.

### 3.4 Warum kein typ-annotierter AST?

In der idealen Welt würde Go einen direkt typ-annotierten AST liefern (wie C# Roslyn oder Java JDT). Das existiert intern im Go-Compiler, ist aber nicht als öffentliche API verfügbar. Die einzige öffentliche Schnittstelle ist `go/types` mit der separaten `types.Info`-Side-Table.

| Sprache            | Ansatz                                                        |
| ------------------ | ------------------------------------------------------------- |
| C# (Roslyn)        | `SemanticModel.GetTypeInfo(node)` – direkt auf jedem Node     |
| Java (Eclipse JDT) | `ITypeBinding` direkt an AST-Nodes                            |
| TypeScript         | `TypeChecker.getTypeAtLocation(node)`                         |
| **Go**             | `types.Info` als separate Hash-Map, AST-Node-Pointer als Keys |

## Quellen

[1] go/types – Config.Check, <https://pkg.go.dev/go/types#Config.Check> </br>
[2] go/types – Info, <https://pkg.go.dev/go/types#Info> </br>
[3] Go Spec – Type unification, <https://go.dev/ref/spec#Type_unification> </br>
[4] Go Spec – Type constraints, <https://go.dev/ref/spec#Type_constraints>
