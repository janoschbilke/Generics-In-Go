# Instantiation Diversity Analysis

## Was wird analysiert?

Die Instantiation-Diversity-Analyse untersucht, **mit welchen konkreten Typ-Argumenten** generische Typen und Funktionen im analysierten Projekt tatsächlich verwendet werden. Ziel ist es, die Vielfalt der Nutzung von Generics zu messen — also nicht nur ob Generics vorhanden sind, sondern wie flexibel sie eingesetzt werden.

Es werden zwei Dimensionen erfasst:

- **Konkrete Instantiierungen**: Der generische Typ/die Funktion wird mit einem echten Typ aufgerufen (z.B. `Box[int]`, `GenericAdd[float64]`)
- **Parametrische Instantiierungen**: Der generische Typ/die Funktion wird innerhalb einer anderen generischen Definition verwendet, wobei der Typ-Parameter weitergereicht wird (z.B. `Box[T]` in einer Methode mit generischem Receiver)

## Wie wird das gemacht?

Die Analyse nutzt Go's `go/types`-Paket mit `types.Info.Instances`. Dieses Map wird beim Type-Checking befüllt und enthält für jeden `*ast.Ident` im AST, der eine generische Instanziierung darstellt, die zugehörigen Typ-Argumente.

Der `InstantiationDiversityCheck` (in `checks/instantiation_checks.go`) iteriert über alle `*ast.Ident`-Knoten im AST und prüft:

1. Ist dieser Identifier in `types.Info.Instances` vorhanden?
2. Ist der zugehörige generische Typ/die Funktion **lokal definiert** (nicht aus einer externen Bibliothek)?
3. Handelt es sich um einen Struct (`*types.Named` mit `*types.Struct` als Underlying) oder eine Funktion (`*types.Signature`)?

Für jede gefundene Instanziierung werden die Typ-Argumente als String gespeichert (z.B. `"int, string"`). Ist eines der Typ-Argumente selbst ein `*types.TypeParam` (also ein Typ-Parameter wie `T`, `K`, `V`), wird die Instanziierung als **parametrisch** markiert.

Die Ergebnisse werden in `context.Instantiations` (`model.InstantiationData`) gesammelt und nach dem Check-Run von `astAnalyzer.go` ausgelesen.

## Ausgabe

Die Ergebnisse werden in zwei Dateien geschrieben:

- **`generic_counters.csv`**: Zählt die Anzahl der **Verwendungsstellen** (call sites) im Code
- **`instantiation_diversity.csv`**: Zählt die Anzahl **eindeutiger Typ-Argument-Kombinationen** pro generischem Typ/Funktion

---

## Ergebnisse LocalTestProject

### First Try (vor Verbesserungen)

Im ersten Durchlauf wurden nur Structs betrachtet, und parametrische Instantiierungen (`T`) wurden nicht gesondert markiert. Sie erschienen demnach gleichwertig neben konkreten Typen wie `int` und `string`. Es ergab sich folgende fragwürdige Ausgabe:

```text
Struct instantiation diversity for local/LocalTestProject:
  Box: 3 unique type combination(s): [T, int, string]
  ComparableContainer: 1 unique type combination(s): [T]
  ContainerWithEmptyInterface: 1 unique type combination(s): [T]
  G: 1 unique type combination(s): [T]
  G2: 1 unique type combination(s): [T]
  NumericContainer: 1 unique type combination(s): [T]
  SimpleContainer: 1 unique type combination(s): [T]
  StorageWithEmptyInterface: 1 unique type combination(s): [T]
  StringableContainer: 1 unique type combination(s): [T]
```

**Probleme des First Try:**

- `T` wurde als gleichwertiger Typ neben `int` und `string` gelistet, obwohl er kein konkreter Typ ist, sondern ein Typ-Parameter, der in einer generischen Methode weitergereicht wird
- Nur Structs wurden betrachtet, generische Funktionen fehlten

### Aktueller Stand (nach Verbesserungen)

```text
Instantiation diversity for local/LocalTestProject:
  [Structs]
    Box: 2 concrete [int, string] + 1 parametric [T]
    ComparableContainer: 1 parametric [T]
    ContainerWithEmptyInterface: 1 parametric [T]
    G: 1 parametric [T]
    G2: 1 parametric [T]
    NumericContainer: 1 parametric [T]
    SimpleContainer: 1 parametric [T]
    StorageWithEmptyInterface: 1 parametric [T]
    StringableContainer: 1 parametric [T]
  [Functions]
    GenericAdd: 2 concrete [float64, int]
    GenericPrint: 2 concrete [int, string]
    makeBox: 2 concrete [int, string]
```

**Verbesserungen:**

- Parametrische Instantiierungen (`T`) werden gesondert ausgewiesen. Sie entstehen durch Methoden mit generischem Receiver und sind keine echten Verwendungsstellen
- Generische Funktionen werden ebenfalls erfasst

---

## Warum unterscheiden sich die Zählungen? (15 vs. 17)

In `generic_counters.csv` stehen für das LocalTestProject:

| Spalte | Wert |
| --- | --- |
| GenericFuncInstantiationExplicit | 4 |
| GenericFuncInstantiationInferred | 6 |
| GenericTypeInstantiationExplicit | 3 |
| GenericTypeInstantiationInferred | 2 |
| **Summe** | **15** |

In `generic_counters_instantiation.csv` ergibt die Summe aller `ConcreteCount + ParametricCount` über alle Zeilen **17**.

Diese Diskrepanz ist **korrekt und erwartet**, weil die beiden Dateien unterschiedliche Dinge messen:

**`generic_counters.csv`** zählt **Verwendungsstellen im Code** — also wie oft ein generischer Typ oder eine generische Funktion im Quellcode tatsächlich aufgerufen/instantiiert wird. Parametrische Instantiierungen (z.B. `Box[T]` in einer Methode mit generischem Receiver) sind **keine echten Verwendungsstellen** und werden hier nicht gezählt.

**`generic_counters_instantiation.csv`** zählt **eindeutige Typ-Argument-Kombinationen** — also wie viele verschiedene Typen für einen generischen Namen beobachtet wurden. Hier werden auch parametrische Einträge mitgezählt, weil sie zeigen, dass ein generischer Typ auch in anderen generischen Kontexten weiterverwendet wird.

Die 17 Diversity-Einträge setzen sich zusammen aus:

- **8 konkrete** eindeutige Kombinationen: `Box[int]`, `Box[string]`, `GenericAdd[float64]`, `GenericAdd[int]`, `GenericPrint[int]`, `GenericPrint[string]`, `makeBox[int]`, `makeBox[string]`
- **9 parametrische** Einträge: `Box[T]` + je `[T]` für die 8 Structs mit generischem Receiver

Die 15 Verwendungsstellen hingegen zählen **wie oft** diese Kombinationen im Code vorkommen. Dabei kann dieselbe Kombination (z.B. `Box[int]`) mehrfach verwendet werden und zählt dann mehrfach. Parametrische Einträge zählen gar nicht mit, da sie keine echten Verwendungsstellen sind.

Kurz: **8 konkrete unique Kombinationen + 9 parametrische Kombinationen = 17 Diversity-Einträge**, während die 15 Verwendungsstellen die tatsächliche Nutzungshäufigkeit im Code widerspiegeln.

## Bezug zur Compiler-Implementierung: Typspezifische Instantiierungen

Generics in Go (wie Templates in C++) sollen **keine zusätzlichen Laufzeitkosten** verursachen, weil der Compiler typspezifische Instantiierungen erzeugt.

### Wie Go Generics kompiliert

Go verwendet eine Hybridstrategie namens **GC Shape Stenciling** (eingeführt mit Go 1.18):

- Für jeden **GC Shape** (grob: Typen mit identischer Speicherrepräsentation und Pointer-Struktur) wird **eine gemeinsame Implementierung** erzeugt
- Konkret: alle Pointer-Typen teilen sich eine Implementierung, alle `int`-artigen Typen mit gleicher Größe teilen sich eine Implementierung usw.
- Zur Laufzeit wird ein **Dictionary** mitgegeben, das typspezifische Informationen (Methodentabellen, Größen) enthält

Das bedeutet: `Box[*int]` und `Box[*string]` teilen sich denselben Maschinencode, aber `Box[int]` und `Box[string]` können separate Implementierungen bekommen, wenn ihre GC Shapes verschieden sind.

### Relevanz für unsere Analyse

Unsere Instantiation-Diversity-Analyse misst genau das, was für den Compiler relevant ist: **wie viele verschiedene Typ-Argument-Kombinationen** für einen generischen Typ/eine Funktion beobachtet werden. Je mehr eindeutige konkrete Kombinationen, desto mehr potenziell separate Compiler-Instantiierungen.

Aus den Ergebnissen des LocalTestProject lässt sich ableiten:

| Generischer Name | Konkrete Kombinationen | Potenzielle Compiler-Instantiierungen |
| --- | --- | --- |
| `Box` | `int`, `string` | 2 (verschiedene GC Shapes) |
| `GenericAdd` | `float64`, `int` | 2 (verschiedene GC Shapes) |
| `GenericPrint` | `int`, `string` | 2 (verschiedene GC Shapes) |
| `makeBox` | `int`, `string` | 2 (verschiedene GC Shapes) |

Die **parametrischen Einträge** (`Box[T]` etc.) erzeugen keine eigenen Compiler-Instantiierungen, da sie nur interne Weiterleitungen innerhalb generischer Definitionen sind.
