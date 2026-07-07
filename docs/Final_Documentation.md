# Generic Analysis Tool

Das Generic Analysis Tool analysiert die Verwendung generischer Funktionen, Methoden und Typen innerhalb eines Go-Packages.
Dabei werden die Anzahl sowie die unterschiedlichen konkreten Instanziierungen in Form ihrer Datentypen ermittelt.

Da der Go-AST ausschließlich die syntaktische Struktur beschreibt, enthält er keine Informationen über inferierte Typen.
Diese Informationen sind erst durch das `go/types` Package verfügbar.
Daher kombiniert das Tool AST-Informationen mit den Typinformationen des Typecheckers, um sowohl explizite als auch inferierte generische Instanziierungen erkennen zu können.

## 1 Kurze Algorithmische Beschreibung

Zur Übersicht folgt eine abstrake Darstellung des Analysevorgangs als Pseudocode.
Dieser wird im nachfolgenden Kapitel anhand eines Beispiels veranschaulicht.

```text
Gegeben ist der AST A eines Packages.

Schritt 1.a
Für jeden Knoten in A: 
    - Speichere jeden Funktionsaufruf und jede Instanziierung (unabhängig davon, ob sie generisch ist).
    - Speichere dessen Typen (via "types").
Speichere diese Informationen in einer Map.

Schritt 1.b
Für jeden Knoten in A: Ermittle ob dieser generisch ist, und speichere ihn in diesem Fall in einer Map.

Schritt 2.a
Baue einen Analysekontext für den AST A auf. Dieser besteht aus:
    - den Typinformationen aus Schritt 1.a
    - den Instanziierungsinformationen aus Schritt 1.a
    - allen generischen AST-Knoten aus Schritt 1.c

Schritt 2.b
Führe Instanziierungs-Checks durch: Für jeden Knoten im AST A
    - Prüfe unter Verwendung der Typ- und Instanziierungsinformationen (1.a), ob diese generisch sind (auf Basis von 1.b), und führe die Informationen zusammen.
        => Wenn eine Instanziierung (1.a) zu einem generischen Typ (1.b) gehört, inferriere den Typ anhand der Typinformationen aus 1.a.
    - Füge den gefundenen Typ der Instanziierungsliste hinzu und erhöhe den Generic-Instanziierungszähler um 1.
```

### 1.1 Algorithmische Beschreibung anhand eines Beispiels

Gegeben seien folgende generische Funktion (`gf`) und die nicht-generische Funktion (`nongf`) und ihre Aufrufe:

```go
func gf[T any](v T) T {
    return v
}

func nongf() { return nil }

func test() {
    x := gf[int](42) // C1 -> explizit
    y := gf("Hello") // C2 -> implizit

    z := nongf() // C3
}
```

```text
Schritt 1.a
Für jeden Knoten in A: 
    - Speichere jeden Funktionsaufruf und jede Instanziierung (unabhängig davon, ob sie generisch ist).
    - Speichere dessen Typen (via "types").
Speichere diese Informationen in einer Map.
```

Für dieses Beispiel werden unter Anderem folgende Werte gespeichert:

| AST-Knoten (ast.Expr-Knoten) | Type |
| --- | --- |
| `gf` (Funktionsidentifizierung, in C1) | `func[T any](v T) T` |
| `gf[int]` (Indizierungsaufruf) | `func(v int) int` |
| `gf[int](42)` (Funktionsaufruf, C1) | `int` |
| `gf` (Funktionsidentifizierung, in C2) | `func[T any](v T) T` |
| `gf("Hello")` (Funktionsaufruf, C2) | `string` |
| `nongf` (Funktionsidentifizierung, in C3) | `func()` |
| `nongf()` (Funktionsaufruf, C3) | `()` |

Weiter speichern wir für die Instanziierungen folgende Informationen:

| Aufruf | `Instances.TypeArgs` | `Instances.Type` |
| --- | --- | --- |
| `gf[int](42)` (C1) | `[int]` | `func(int) int` |
| `gf("Hello")` (C2) | *__`[string]`__* | `func(string) string` |
| `nongf()` (C3) | `` | `func(void) void` |

An dieser Stelle erkennt man an der markierten *`[string]`* Angabe, dass hier die tatsächliche Inferrierung passiert.

```text
Schritt 1.b
Für jeden Knoten in A: Ermittle ob dieser generisch ist, und speichere ihn in diesem Fall in einer Map.
```

Diese Tabelle hat bei uns nur einen Eintrag, weil wir nur eine generische Funktion haben:

| Generischer Knoten | Typparameter |
| ------------------ | ------------ |
| `gf`               | `T`          |

```text
Schritt 2.a
Baue einen Analysekontext für den AST A auf. Dieser besteht aus:
    - den Typinformationen aus Schritt 1.a
    - den Instanziierungsinformationen aus Schritt 1.a
    - allen generischen AST-Knoten aus Schritt 1.c
```

Der Analysekontext besteht aus

- Typinformationen (Schritt 1.a)
- Instanziierungsinformationen (Schritt 1.a)
- generischen AST-Knoten (Schritt 1.b)

```text
Schritt 2.b
Führe Instanziierungs-Checks durch: Für jeden Knoten im AST A
    - Prüfe unter Verwendung der Typ- und Instanziierungsinformationen (1.a), ob diese generisch sind (auf Basis von 1.b), und führe die Informationen zusammen.
        => Wenn eine Instanziierung (1.a) zu einem generischen Typ (1.b) gehört, inferriere den Typ anhand der Typinformationen aus 1.a.
    - Füge den gefundenen Typ der Instanziierungsliste hinzu und erhöhe den Generic-Instanziierungszähler um 1.
```

Für `C1: gf[int](42)` gilt: Explizite Aufrufe können direkt syntaktisch aus dem AST gewonnen werden.

Für `C2: gf("Hello")` gilt: Für diesen impliziten Aufruf muss in die Typ- und Instanziierungstabellen geschaut werden.
Über die Typinformationen aus Schritt 1.a wird der Typ des Arguments bestimmt (siehe markierte Zelle in der Instanziierungstabelle).

Für `C3: nongf()` gilt: Sie befindet sich nicht in der Generic-Map. Es liegt daher keine generische Instantiierung vor.

Schlussendlich wird der Zähler `genericInstantiationCounter` um 2 erhöht. Die Instanziierungsliste besteht final aus `[int, string]`.

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

__Fetch dominiert die Gesamtlaufzeit.__
Im GitHub-Modus macht `fetch_files_ms` bei den meisten Repos 70–95 % der Gesamtlaufzeit aus. Für `kubernetes/kubernetes` sind es 8,7 s von 12,6 s Gesamtzeit. Die eigentliche Analyse (3,9 s) ist schneller als das Laden der Dateien. Das liegt daran, dass für jede Datei ein separater GitHub-API-Aufruf nötig ist.

__`type_check_ms` ist die teuerste Analyse-Phase.__
Innerhalb von `total_analysis_ms` entfallen typischerweise ~50 % auf `type_check_ms`, ~30 % auf `parse_ast_ms` und ~20 % auf `check_runner_ms`. `topo_sort_ms` und `collect_symbols_ms` sind in allen Fällen vernachlässigbar (< 5 % von `total_analysis_ms`).

__Repos ohne Generics: Analyse < 1 ms.__
Projekte wie `joesonw/go-diff` (3 Funktionen, keine Generics) zeigen 0 ms für alle Analyse-Phasen. Das ist kein Messfehler, sondern ein Auflösungsproblem der Millisekunden-Granularität, die Analyse läuft in unter 1 ms durch.

__Korrelation zwischen Projektgröße und `type_check_ms`.__
`kubernetes/kubernetes` hat 50.884 Funktionen und braucht 1.951 ms für `type_check`. `junegunn/fzf` hat 403 Funktionen und braucht 35 ms. Das Verhältnis (~56×) entspricht grob dem Verhältnis der Funktionsanzahl (~126×), was zeigt, dass `types.Check` annähernd linear mit der Paketgröße skaliert.

---

## 3. Wie `types.Check` funktioniert

### 3.1 Was `types.Check` analysiert

`types.Config.Check(importPath, fset, files, info)` führt für die übergebenen Dateien einen vollständigen semantischen Analyse-Pass durch:

1. __Name Resolution__ – Alle Bezeichner werden ihren Deklarationen zugeordnet [2]
2. __Type Inference__ – Typ-Parameter werden aus dem Kontext inferiert [3]
3. __Constraint Checking__ – Typ-Argumente werden gegen ihre Constraints geprüft [4]
4. __Instantiation Recording__ – Alle generischen Instanziierungen werden in `info.Instances` eingetragen [2]

### 3.2 Analysiert `Check` immer das gesamte Paket?

__Ja.__ `types.Check` analysiert immer alle übergebenen Dateien als Einheit. Es gibt keine Möglichkeit, nur einen einzelnen AST-Knoten zu checken. Selbst wenn man nur an einer einzigen Instanziierung interessiert ist, muss das gesamte Paket gecheckt werden.

```text
Naive Implementierung (alter Ansatz):
    for each file:
        types.Check([file])     // N_files Aufrufe, jede Datei isoliert

Optimierte Implementierung (neuer Ansatz):
    for each package (in topo. Reihenfolge):
        types.Check(package.allFiles)   // N_packages Aufrufe, ganzes Paket
```

Der Unterschied: Bei einem Paket mit 50 Dateien reduziert sich die Anzahl der `types.Check`-Aufrufe von 50 auf 1. Außerdem kann der Type-Checker im neuen Ansatz Typen über Dateigrenzen hinweg auflösen.

__Formale Betrachtung als Mengen:__

Sei P ein Go-Paket mit Dateien F = {f_1, ..., f_m}. Jede Datei f_i wird zu einem AST mit einer Menge von Knoten N_i geparst. Die Gesamtmenge aller Knoten, die `types.Check` übergeben wird:

N = N_1 ∪ N_2 ∪ ... ∪ N_m = {n_1, ..., n_k}

Innerhalb von N unterscheiden wir:

- __G ⊆ N__ – generische Definitionen (Funktionen/Typen mit Typ-Parametern)
- __I ⊆ N__ – Instanziierungsstellen (Call-Expressions, Composite Literals)
- __R = N \ (G ∪ I)__ – alle übrigen nicht-generischen Knoten

Die naive Annahme wäre, dass `types.Check` nur G ∪ I verarbeiten muss. Die Realität: `types.Check` verarbeitet immer die gesamte Menge N, unabhängig von |G| und |I|. Der Benchmark in Section 3.3 bestätigt dies empirisch: Laufzeit ∝ |N|, nicht ∝ |G| oder |I|.

### 3.3 Benchmark: Skaliert `Check` mit Dateigröße oder Instanziierungsanzahl?

Um zu verstehen, womit `types.Check` skaliert, wurde ein Mikrobenchmark mit einer 2×2-Matrix durchgeführt (Quellcode: `docs/assets/typecheck_benchmark/`). Synthetisch generierte Go-Dateien variieren unabhängig voneinander in Dateigröße (Anzahl nicht-generischer Funktionen) und Anzahl generischer Instanziierungen.

Das `types.Info`-Struct im Benchmark entspricht dem echten Analyzer (`Uses`, `Types`, `Instances` alle non-nil). Die offizielle Dokumentation erklärt warum das relevant ist:

> *"Check populates each of the non-nil maps in the Info struct."* [1]

Jede non-nil Map bedeutet zusätzliche Arbeit für `Check`. Der Benchmark misst daher die reale Laufzeit des Analyzers, nicht eine vereinfachte Variante.

__Ergebnisse__ (Apple M4 Pro, Go 1.22, `go test -bench=. -benchtime=3s`):

| Szenario                                | Nicht-generische Fns | Generische Fns | Instanziierungen | ns/op     | Faktor   |
| --------------------------------------- | -------------------- | -------------- | ---------------- | --------- | -------- |
| Baseline                                | 0                    | 1              | 1                | 5.376     | 1×       |
| Viele Instanziierungen                  | 0                    | 1              | 1.000            | 2.447.963 | __455×__ |
| Viele nicht-generische Fns, 1 Inst.     | 1.000                | 1              | 1                | 2.175.650 | __405×__ |
| Viele nicht-generische Fns, viele Inst. | 1.000                | 1              | 1.000            | 5.054.883 | __940×__ |
| Viele generische Fns, 1 Inst.           | 0                    | 1.000          | 1                | 2.134.096 | __397×__ |
| Viele generische Fns, viele Inst.       | 0                    | 1.000          | 1.000            | 4.635.733 | __862×__ |

__Interpretation:__

`types.Check` skaliert mit __allen drei__ Dimensionen. Die entscheidende Erkenntnis liefert der Vergleich der letzten vier Zeilen:

- 1.000 nicht-generische Fns + 1 Inst. → __2.175.650 ns__
- 1.000 generische Fns + 1 Inst. → __2.134.096 ns__

Diese beiden Werte sind nahezu identisch (~2% Unterschied). `types.Check` unterscheidet __nicht__ zwischen generischen und nicht-generischen Funktionen – es verarbeitet alle Deklarationen im File, unabhängig davon ob sie generisch sind oder ob sie überhaupt aufgerufen werden. Die 999 nicht-aufgerufenen generischen Funktionen werden trotzdem vollständig gecheckt.

Es gibt keine "intelligente" Abkürzung: Selbst wenn nur eine einzige generische Funktion im gesamten File instantiiert wird, muss `Check` alle anderen Funktionen ebenfalls auflösen, typisieren und in die `Uses`- und `Types`-Maps eintragen.

### 3.4 Warum kein typ-annotierter AST?

In der idealen Welt würde Go einen direkt typ-annotierten AST liefern (wie C# Roslyn oder Java JDT). Das existiert intern im Go-Compiler, ist aber nicht als öffentliche API verfügbar. Die einzige öffentliche Schnittstelle ist `go/types` mit der separaten `types.Info`-Side-Table.

| Sprache            | Ansatz                                                        |
| ------------------ | ------------------------------------------------------------- |
| C# (Roslyn)        | `SemanticModel.GetTypeInfo(node)` – direkt auf jedem Node     |
| Java (Eclipse JDT) | `ITypeBinding` direkt an AST-Nodes                            |
| TypeScript         | `TypeChecker.getTypeAtLocation(node)`                         |
| __Go__             | `types.Info` als separate Hash-Map, AST-Node-Pointer als Keys |

## Quellen

[1] go/types – Config.Check, <https://pkg.go.dev/go/types#Config.Check> </br>
[2] go/types – Info, <https://pkg.go.dev/go/types#Info> </br>
[3] Go Spec – Type unification, <https://go.dev/ref/spec#Type_unification> </br>
[4] Go Spec – Type constraints, <https://go.dev/ref/spec#Type_constraints>
