# Generic Analysis Tool

## 1. Analyse-Walkthrough anhand eines konkreten Beispiels

### 1.1 Das Beispiel

Gegeben sei ein kleines Go-Projekt mit zwei Dateien in zwei Paketen:

```go
// collections/stack.go
package collections

type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T {
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

```
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

```
Reihenfolge: ["collections", "."]
```

`collections` wird zuerst gecheckt, damit `main` es als Dependency auflösen kann.

---

### 1.5 Schritt 4: `extractTypeInfoForPackages` – N × `types.Check`

Für jedes Paket in topologischer Reihenfolge wird `types.Config.Check()` aufgerufen.

**Aufruf 1: `types.Check` für Paket `collections`**

```
Input:  [stack.go AST]
Output: types.Info{
    Instances: {}   // keine Instanziierungen in der Definition selbst
}
typedPkg -> wird in projectImporter gespeichert
```

**Aufruf 2: `types.Check` für Paket `main`**

```
Input:  [main.go AST], Importer kennt bereits "collections"
Output: types.Info{
    Instances: {
        Ident("Stack") -> Instance{ TypeArgs: [int] }
        Ident("Push")  -> Instance{ TypeArgs: [int] }
        Ident("Pop")   -> Instance{ TypeArgs: [int] }
    }
}
```

Genau hier liegt der Kern: `types.Check` inferiert, dass `s.Push(42)` eine Instanziierung von `Push[int]` ist, obwohl `[int]` im Quellcode nicht steht. Das ist nur möglich, weil `s` als `Stack[int]` bekannt ist – und das wiederum nur, weil `collections` bereits gecheckt und im `projectImporter` verfügbar ist.

---

### 1.6 Schritt 5: `collectProjectSymbols`

Für alle gecheckte Pakete wird `CollectAll` aufgerufen (ein einziger `ast.Inspect`-Walk pro Datei), der drei Maps gleichzeitig befüllt:

```
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

```
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

| Phase | Feld | Beschreibung |
|---|---|---|
| Fetch Files | `fetch_files_ms` | GitHub API-Aufrufe oder lokales FS-Lesen |
| Parse AST | `parse_ast_ms` | `parser.ParseFile` für alle Dateien aller Pakete |
| Topo Sort | `topo_sort_ms` | Topologische Sortierung der Paket-Abhängigkeiten |
| Type Check | `type_check_ms` | N_packages × `types.Config.Check()` |
| Collect Symbols | `collect_symbols_ms` | Aufbau der projekt-weiten Generic-Maps |
| Check Runner | `check_runner_ms` | Alle AST-Checks über alle Dateien |
| Total Analysis | `total_analysis_ms` | Summe aller Analyse-Phasen (ohne Fetch) |
| Total | `total_ms` | Fetch + Total Analysis |

### 2.2 Ergebnisse (Auswahl aus 100 analysierten Repos)

| Repository | fetch_ms | parse_ast_ms | type_check_ms | check_runner_ms | total_analysis_ms | total_ms | Generics* |
|---|---|---|---|---|---|---|---|
| kubernetes/kubernetes | 8729 | 1324 | 1951 | 478 | 3895 | 12625 | 271 |
| ollama/ollama | 3209 | 116 | 192 | 40 | 361 | 3571 | 35 |
| jesseduffield/lazygit | 1525 | 53 | 115 | 22 | 196 | 1721 | 22 |
| gohugoio/hugo | 2312 | 77 | 111 | 28 | 225 | 2538 | 27 |
| syncthing/syncthing | 1437 | 57 | 83 | 18 | 164 | 1602 | 16 |
| netdata/netdata | 3807 | 202 | 308 | 68 | 600 | 4408 | 46 |
| junegunn/fzf | 667 | 25 | 35 | 6 | 69 | 736 | 2 |
| anthdm/hbbft | 725 | 4 | 5 | 0 | 11 | 736 | 0 |
| joesonw/go-diff | 640 | 0 | 0 | 0 | 0 | 640 | 0 |

*`FuncGeneric` aus `generic_counters.csv`

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

1. **Name Resolution** – Alle Bezeichner werden ihren Deklarationen zugeordnet
2. **Type Inference** – Typ-Parameter werden aus dem Kontext inferiert
3. **Constraint Checking** – Typ-Argumente werden gegen ihre Constraints geprüft
4. **Instantiation Recording** – Alle generischen Instantiierungen werden in `info.Instances` eingetragen

### 3.2 Analysiert `Check` immer das gesamte Paket?

**Ja.** `types.Check` analysiert immer alle übergebenen Dateien als Einheit. Es gibt keine Möglichkeit, nur einen einzelnen AST-Knoten zu checken. Selbst wenn man nur an einer einzigen Instantiierung interessiert ist, muss das gesamte Paket gecheckt werden.

```
Naive Implementierung (alter Ansatz):
    for each file:
        types.Check([file])     // N_files Aufrufe, jede Datei isoliert

Optimierte Implementierung (neuer Ansatz):
    for each package (in topo. Reihenfolge):
        types.Check(package.allFiles)   // N_packages Aufrufe, ganzes Paket
```

Der Unterschied: Bei einem Paket mit 50 Dateien reduziert sich die Anzahl der `types.Check`-Aufrufe von 50 auf 1. Außerdem kann der Type-Checker im neuen Ansatz Typen über Dateigrenzen hinweg auflösen.

### 3.3 Warum kein typ-annotierter AST?

In der idealen Welt würde Go einen direkt typ-annotierten AST liefern (wie C# Roslyn oder Java JDT). Das existiert intern im Go-Compiler, ist aber nicht als öffentliche API verfügbar. Die einzige öffentliche Schnittstelle ist `go/types` mit der separaten `types.Info`-Side-Table.

| Sprache | Ansatz |
|---|---|
| C# (Roslyn) | `SemanticModel.GetTypeInfo(node)` – direkt auf jedem Node |
| Java (Eclipse JDT) | `ITypeBinding` direkt an AST-Nodes |
| TypeScript | `TypeChecker.getTypeAtLocation(node)` |
| **Go** | `types.Info` als separate Hash-Map, AST-Node-Pointer als Keys |