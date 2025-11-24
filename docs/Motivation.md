# Generics-In-Go

- [Generics-In-Go](#generics-in-go)
  - [Grundlagen Generics in Go](#grundlagen-generics-in-go)
    - [Generische Funktionssignaturen und  Type Parameters](#generische-funktionssignaturen-und--type-parameters)
    - [Constraints und Type Sets](#constraints-und-type-sets)
    - [Type Inference](#type-inference)
    - [Limitationen](#limitationen)
      - [Kein Support für polymorphe Rekursion](#kein-support-für-polymorphe-rekursion)
      - [Code-Bloat und Kompilierzeiten](#code-bloat-und-kompilierzeiten)
      - [Eingeschränkte Constraints und Expressivität](#eingeschränkte-constraints-und-expressivität)
      - [Weitere beachtenswerte Punkte](#weitere-beachtenswerte-punkte)
  - [Grobflächige Analyse](#grobflächige-analyse)
    - [Anmerkung zu RegEx und Motivation für einen GoParser](#anmerkung-zu-regex-und-motivation-für-einen-goparser)
  - [GoParser](#goparser)
    - [Funktionsweise](#funktionsweise)
    - [Anleitung](#anleitung)
    - [Ergebnisse](#ergebnisse)
  - [Verwendung von Large Language Models zur Analyse von Generics in Go](#verwendung-von-large-language-models-zur-analyse-von-generics-in-go)
  - [Fazit](#fazit)
  - [Ausblick](#ausblick)
  - [Quellenverzeichnis](#quellenverzeichnis)


## Grundlagen Generics in Go

Generics in Go ermöglichen es, Code zu schreiben, der unabhängig von spezifischen Datentypen funktioniert. Sie wurden mit Go 1.18 eingeführt und stellen eine der bedeutendsten Ergänzungen zur Sprache dar, seit ihrer ersten Veröffentlichung. Im Kern erlauben Generics die Definition von Funktionen und Typen, die mit einer Vielzahl von Typen arbeiten können, ohne dass der Code für jeden Typ separat geschrieben werden muss. Dies fördert die Wiederverwendbarkeit und reduziert Redundanz.

Im Vergleich zu Sprachen wie Java funktionieren Generics in Go grundlegend anders. In Java werden Generics durch Type Erasure umgesetzt: Zur Kompilierzeit werden Typinformationen überprüft, aber zur Laufzeit werden sie entfernt, was zu einer einheitlichen Bytecode-Repräsentation führt. Das kann zu Laufzeitfehlern führen und schränkt die Reflexion ein. Go hingegen instantiiert Generics zur Kompilierzeit vollständig, was bedeutet, dass für jede Typkombination spezialisierter Code generiert wird. Das führt zu besserer Laufzeitperformance, da keine Typüberprüfungen zur Laufzeit nötig sind, birgt aber das Risiko von Code-Bloat (vermehrte Code-Duplikation) und längeren Kompilierzeiten. Go verwendet Constraints, um die erlaubten Typen einzuschränken, was eine statische Typprüfung ermöglicht. Ein weiterer wesentlicher Unterschied liegt im Subtyping: Java basiert auf nominalem Subtyping, bei dem Typkompatibilität explizit deklariert werden muss (z. B. durch extends), was Generics invariant macht und zusätzliche Konstrukte wie Wildcards erfordert. Go hingegen nutzt strukturelles Subtyping, bei dem Typen automatisch kompatibel sind, wenn sie die gleiche Struktur (z. B. Methoden) aufweisen. Dies macht Constraints flexibler und reduziert die Notwendigkeit für Typ-Hierarchien.

Im Folgenden werden die Kernkonzepte von Generics in Go erläutert, ergänzt um praktische Beispiele. Wir beginnen mit den Type Parameters, gehen dann auf Constraints und Type Sets ein und schließen mit Type Inference ab.

### Generische Funktionssignaturen und  Type Parameters

Type Parameters erlauben es, Funktionen oder Typen mit variablen Typen zu parametrieren. Sie werden in eckigen Klammern angegeben, ähnlich wie normale Parameter in runden Klammern.

Betrachten wir eine einfache nicht-generische Funktion, die das Minimum zweier Ganzzahlen berechnet:

```go
func MinInt(x, y int) int {
    if x < y {
        return x
    }
    return y
}
```

Um diese Funktion generisch zu machen, fügen wir einen Type Parameter `T` hinzu und ersetzen `int` durch `T`. Wir müssen jedoch sicherstellen, dass `T` vergleichbar ist. Hier ein Beispiel mit einer Constraint für vergleichbare Typen:

```go
import "golang.org/x/exp/constraints"

func GenericMin[T constraints.Ordered](x, y T) T {
    if x < y {
        return x
    }
    return y
}
```

Diese Funktion kann nun mit verschiedenen Typen instantiiert werden, z. B.:

```go
result := GenericMin[float64](3.14, 2.71) // Ergibt 2.71
```

Type Parameters können auch für benutzerdefinierte Typen verwendet werden. Ein Beispiel für einen generischen Stack:

```go
type Stack[T any] struct {
    items []T
}

func (s *Stack[T]) Push(item T) {
    s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() T {
    if len(s.items) == 0 {
        panic("Stack ist leer")
    }
    item := s.items[len(s.items)-1]
    s.items = s.items[:len(s.items)-1]
    return item
}
```

Hier speichert der Stack Werte eines beliebigen Typs `T`. Eine Instanzierung könnte so aussehen:

```go
var intStack Stack[int]
intStack.Push(42)
```

### Constraints und Type Sets

Constraints definieren, welche Typen für einen Type Parameter erlaubt sind. In Go werden Constraints als Interfaces dargestellt, die nicht nur Methoden, sondern auch Typmengen (Type Sets) beschreiben können. Das erlaubt es, Typen basierend auf ihren Eigenschaften einzuschränken, z. B. ob sie vergleichbar oder arithmetisch operierbar sind.

Ein Beispiel für eine Constraint ist `constraints.Ordered`, die alle Typen umfasst, die mit `<` verglichen werden können:

```go
type Ordered interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 | // Und weitere Integer-Typen
    ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
    ~float32 | ~float64 |
    ~string
}
```

Der Tilde-Operator `~` schließt abgeleitete Typen ein, z. B. `type MyInt int` für `~int`. Unions (`|`) kombinieren Typen.

In einer Funktion könnte eine solche Constraint so verwendet werden:

```go
func Max[T ~int | ~float64](a, b T) T {
    if a > b {
        return a
    }
    return b
}
```

Dies erlaubt Aufrufe wie `Max(5, 3)` oder `Max(4.2, 1.1)`, aber nicht mit Strings.

Interfaces als Type Sets erweitern die Flexibilität: Ein Interface kann nun Typen explizit auflisten oder Methoden fordern, was Go's Typensystem bereichert.

### Type Inference

Type Inference ermöglicht es, Type Arguments bei Funktionsaufrufen zu omitieren, wenn der Compiler sie aus den Argumenten ableiten kann. Das macht den Code lesbarer und näher an nicht-generischem Code.

Es gibt zwei Formen: Function Argument Type Inference (ableitet aus Funktionsparametern) und Constraint Type Inference (ableitet aus Constraints).

Beispiel für eine Skalierungsfunktion für Vektoren:

```go
func Scale[S ~[]E, E constraints.Real](s S, factor E) S {
    result := make(S, len(s))
    for i, v := range s {
        result[i] = v * factor
    }
    return result
}
```

Aufruf ohne explizite Typen:

```go
type Vector []float64
v := Vector{1.0, 2.0}
scaled := Scale(v, 2.0) // Inferiert S=Vector, E=float64
```

Hier leitet der Compiler `E` aus dem Constraint `~[]E` ab. Wenn Inference fehlschlägt, muss man Typen explizit angeben, z. B. `Scale[Vector, float64](v, 2.0)`.

### Limitationen

Trotz ihrer Nützlichkeit haben Generics in Go einige bewusste Einschränkungen, die aus der Design-Philosophie der Sprache von Einfachheit und Performanz resultieren. Basierend auf Analysen, wie beispielsweise [1](#quellenverzeichnis), lassen sich folgende Limitationen identifizieren. Diese ergeben sich teilweise aus der Hybrid-Implementierung von Go, die Monomorphisierung (Spezialisierung zur Compile-Zeit) mit Dictionary-Passing (Weitergabe von Typinformationen zur Laufzeit) kombiniert.

#### Kein Support für polymorphe Rekursion

Go kann Programme mit bestimmten rekursiven Typ-Instantiierungen nicht verarbeiten. Der Grund dafür ist, dass der bei der Kompilierung stattfindende Prozess der Monomorphisierung bei bestimmten Mustern in eine unendliche Schleife geraten würde. Ein klassisches Beispiel ist eine rekursive Datenstruktur, die sich selbst mit einem variierenden Typ-Argument instantiieren würde:

```go
// Führt zu unendlicher Rekursion bei der Instantiierung durch den Compiler
type RecursiveList[T any] struct {
    Head T
    Tail *RecursiveList[RecursiveList[T]]
}
```
Dieser Code würde beim Versuch der Kompilierung fehlschlagen, da der Compiler die unendliche Instantiierungssequenz nicht auflösen kann. Ansätze, die rein auf Dictionary-Passing basieren, könnten solche Fälle besser handhaben.

#### Code-Bloat und Kompilierzeiten

Die Spezialisierung für jeden verwendeten Typ kann zu dupliziertem Code führen, was die Größe der resultierenden Binärdatei erhöht ("Code Bloat") und die Kompilierungszeiten verlangsamt. Benchmarks in verwandten Forschungsarbeiten zeigten, dass reine Monomorphisierungs-Übersetzer (wie in Go 1.18) deutlich langsamere Kompilierzeiten aufweisen können als nicht-spezialisierende Methoden. Das Go-Team arbeitet jedoch kontinuierlich an Verbesserungen, um diesen Effekt zu minimieren.

Beispiel: Eine generische Funktion, die mit vielen verschiedenen Typen aufgerufen wird, erzeugt multiple, spezialisierte Versionen ihres eigenen Codes:
```go
func Process[T any](data []T) { /* ... */ }

// Aufrufe mit int, float64, string etc. führen potenziell zu separaten Implementierungen im Binärcode.
```

#### Eingeschränkte Constraints und Expressivität

Die Ausdruckskraft von Go-Generics ist in mehreren Bereichen bewusst begrenzt, um die Komplexität niedrig zu halten.

**1. Constraints auf Methoden-Empfängern:**
Eine der signifikantesten Einschränkungen ist, dass Methoden keine eigenen, zusätzlichen Typ-Parameter einführen können [2](#quellenverzeichnis) [3](#quellenverzeichnis). Außerdem können die Typ-Parameter eines generischen Typs in einer Methode nicht weiter eingeschränkt (spezialisiert) werden. Alle Constraints müssen bereits in der Typ-Definition festgelegt sein.

Dies erzwingt ein weniger flexibles Design. Soll eine Methode für eine generische Struktur nur unter bestimmten Bedingungen verfügbar sein (z. B. wenn der Typ `T` druckbar ist), muss diese Bedingung bereits in die Definition der Struktur selbst aufgenommen werden:

```go
// WENIGER FLEXIBEL: Constraint wird für alle Instanzen von 'Pair' benötigt
type Pair[T Show, S Show] struct {
    Left  T
    Right S
}

// Methode 'Show' ist immer verfügbar, da die Constraints im Typ definiert sind.
func (p Pair[T, S]) Show() string {
    return "(" + p.Left.Show() + "," + p.Right.Show() + ")"
}
```

Ein flexiblerer Ansatz, bei dem die Constraints erst in der Methode definiert werden, ist in Go nicht erlaubt:
```go
type FlexiblePair[T any, S any] struct {
    Left  T
    Right S
}

// UNGÜLTIG: Constraints können nicht im Methoden-Empfänger spezialisiert werden.
/*
func (p FlexiblePair[T Show, S Show]) Show() string {
    return "(" + p.Left.Show() + "," + p.Right.Show() + ")"
}
*/
```
Interessanterweise ist dieses Muster für eigenständige Funktionen gültig, was die Design-Entscheidung für Methoden uneinheitlich erscheinen lässt.

**2. Keine Typ-Zusicherungen (Type Assertions) für Typ-Parameter:**
Es ist nicht möglich, eine Typ-Zusicherung direkt auf einer Variable auszuführen, deren Typ ein Typ-Parameter ist. Der Compiler verhindert dies, da zur Compile-Zeit der Typ-Parameter für einen einzigen, vom Aufrufer bereitgestellten Typ steht, aber nicht als Interface-Typ behandelt wird, auf dem Typ-Zusicherungen typischerweise operieren [7].

```go
// UNGÜLTIG: Typ-Zusicherung auf dem Typ-Parameter 'T'
func process[T any](value T) {
    // Führt zu: "invalid operation: cannot use type assertion on type parameter value"
    specificValue := value.(MyType) 
    // ...
}
```
Der Workaround besteht darin, den Wert zuerst in `any` (oder `interface{}`) zu konvertieren und die Zusicherung dann darauf auszuführen.
```go
// GÜLTIG:
func process[T any](value T) {
    specificValue := any(value).(MyType)
    // ...
}
```
Diese Einschränkung wurde bewusst in das Sprachdesign aufgenommen, um Verwirrung zu vermeiden, erfordert jedoch einen expliziten Zwischenschritt.
[4](#quellenverzeichnis)
**3. Weitere Einschränkungen bei Constraints:**
*   **Keine variadischen Typ-Parameter:** Es ist nicht möglich, eine generische Funktion mit einer variablen Anzahl von Typ-Parametern zu definieren.
*   **Keine Operator-Überladung:** Constraints können zwar das Vorhandensein von Operatoren wie `+` oder `==` fordern (z.B. über `comparable`), aber es gibt keine Möglichkeit, das Verhalten dieser Operatoren für benutzerdefinierte Typen zu definieren.

#### Weitere beachtenswerte Punkte

*   **Begrenzte GC-Shape-Unterstützung:** Frühe Versionen von Go-Generics (z.B. 1.18) betrachteten für die Wiederverwendung von generischem Code durch den Garbage Collector hauptsächlich, ob ein Typ ein Pointer ist. Dies schränkt die Effizienz der Dictionary-Nutzung und Code-Größenreduktion ein.
*   **Schwache Typinferenz:** In komplexeren Fällen, insbesondere wenn ein Typ-Parameter nur in der Rückgabeposition einer Funktion vorkommt, kann der Compiler den Typ nicht immer selbst herleiten, was explizite Typ-Argumente beim Aufruf erfordert 
[5](#quellenverzeichnis).
Diese Limitationen machen Go-Generics robust und pragmatisch, aber weniger expressiv als in Sprachen wie Rust, C++ oder Haskell [6](#quellenverzeichnis). Die Designentscheidungen zielen darauf ab, die Komplexität des Typsystems zu beherrschen und die charakteristische schnelle Kompilierzeit und Einfachheit von Go so weit wie möglich zu erhalten.

## Grobflächige Analyse
Es wurden zunächst verschiedene Strategieen untersucht, um eine grobe Analyse zur Verwendung von Generics in Go anzustellen. Unteranderem wurden rudimentäre Google Suche und die Github-Suchfunktion verwendet. Mit diesen Suchfunktionen konnte jedoch keine sinnvolle Erkenntnis erlangt werden. Weiterhin wurde ein Haskell-Parser in Erwägung gezogen, allerdings nicht verwendet, da dieser initial nicht in der Lage war die Anforderungen für eine Codeanalyse zu Generics in Go zu bewältigen und die Erweiterung des Parsers den Zeitrahmen dieser Arbeit sprengen würde. Es wurde daher zunächst ein Ansatz mit Sourcegraph aus einer vorherigen Masterarbeit gewählt, mit dem Ziel diesen zu verfeinern.
In der ursprünglichen Masterarbiet wird Sourcegraph als zentrales Tool vorgestellt, das durch seine fortschrittlichen Suchfunktionen – insbesondere die Unterstützung regulärer Ausdrücke – eine effiziente Analyse von Quellcode über große Repositories hinweg ermöglicht. Diese Arbeit erweitert diese Ansätze, indem sie die entwickelten regulären Ausdrücke und Suchstrategien überprüft und die Daten auffrischt.

Sourcegraph ist eine leistungsstarke Plattform für Codesuche und -Navigation, die Entwicklern hilft, Quellcode über Repositories und Sprachen hinweg zu durchsuchen und Abhängigkeiten zu identifizieren. Als kommerzielles Produkt für Unternehmen kann es selbst gehostet werden, um eigene Codebasen zu indexieren. Eine kostenlose öffentliche Instanz indexiert populäre GitHub-Repos (basierend auf Sternen), die über eine Million Repos umfasst. Die Syntax ähnelt der GitHub-Suche, bietet aber erweiterte Funktionen und ermöglicht die Indexierung alternativer Plattformen. Der Rechercheansatz nutzt reguläre Ausdrücke, um generische Go-Konstrukte (z. B. nach Go-Benennungsregeln) zu suchen. Für verschiedene Arten von Generics in Go wurden spezifische RegEx erstellt, ergänzt durch eine Beispielsuche nach allen Go-Projekten via Sourcegraph-Query (`context:global language:Go select:repo count:all`), die Repos statt einzelner Codestellen auflistet und alle Ergebnisse liefert. Die folgenden regulären Ausdrücke dienen der Suche nach den bereits oben genannte Arten von Generics in Go und sind so wie die folgenden Beispiele teilweise direkt aus der vorherigen Arbeit übernommen. Die Suche wurde dabei über die Suchleiste der sourcegraph Website durchgeführt, wobei Forks und archivierte Repos nicht miteinbezogen wurden.


**Suche nach generischen Funktionssignaturen:**  
Beispiel:
```go
func PickRandom[T any](choices []ChoiceWeight[T]) T {
	...
}
```
Ursprünglicher (fehlerhafter) RegEx:
```
/func\s*?(\(.+?\))?\s*?[a-zA-Z_]\w*?\s*?\[.+?\]\s*?\(.*?\).*?\{/
```
Dieser reguläre Ausdruck wurde angepasst, um false positives auszuschließen. Zum Beispiel würde dieser RegEx auf folgende Funktionssignatur anspringen, obwohl es sich hierbei um einen Array und nicht um eine genereische Funktion handelt:
```go
func InvalidGeneric[123](param string) error {
    fmt.Println(param)
    return nil
}
```
Es wurde folglich ein robusterer und strengerer regulärer Ausdruck entwickelt, um mehr false positives auszuschließen.

Neuer, robuster RegEx:
```
/func\s*?(\(.+?\))?\s*?[a-zA-Z_]\w*?\s*?\[\s*?[a-zA-Z_]\w*?\s*?(?:,\s*?[a-zA-Z_]\w*?)*\s*?(?:any|comparable|interface\s*?\{.*?\}|~[a-zA-Z_]\w*?(?:\s*?\|\s*?~?[a-zA-Z_]\w*?)*)\s*?\]\s*?\(.*?\).*?\{/
```

**Suche nach Deklarationen von Typ Parametern:**  
Beispiel:
```go
type Tree[T interface{}] struct {
    left, right *Tree[T]
    value       T
}
```
Ursprünglicher (fehlerhafter) RegEx:
```
/type\s+?[a-zA-Z_]\w*?\s*?\[.+?\]\s+?(struct|interface)\s*?\{/
```

Auch dieser RegEx leidet unter dem Problem, dass er auf Arrays anspringt. Ein beispielhaftes false positive wäre somit:
```go
// False Positive: Array mit fester Größe, keine Generics
type FixedArray[10] struct {
    data [10]int
}
```
Um dies in Zukunft vorzubeugen wurde dieser neue, robustere RegEx entwickelt, welcher nicht mehr auf die aufgezeigten false positives anschlägt.

Neuer RegEx:
```
/type\s+?[a-zA-Z_]\w*?\s*?\[\s*?[a-zA-Z_]\w*?\s*?(?:,\s*?[a-zA-Z_]\w*?)*\s*?(?:any|comparable|interface\s*?\{.*?\}|~[a-zA-Z_]\w*?(?:\s*?\|\s*?~?[a-zA-Z_]\w*?)*)\s*?\]\s+?(struct|interface)\s*?\{/
```

**Suche nach Deklaration von Type Sets**:  
Beispiel:
```go
type Ordered interface {
    Integer|Float|~string
}
```
ursprünglicher (fehlerhafter) RegEx:
```
/type\s+?[a-zA-Z_]\w*?\s+?interface\s*?{(\n)?[^|}]*\|.*?(\n)?}/
```

Auch dieser RegEx schlägt auf einige false positives an, wie zum Beispiel bei dieser Typdeklaration:
```go
// False Positive: Interface mit Methode, die `|` in der Signatur hat
type BitOperation interface {
    BitOr(a, b int) (result int | nil) // Rückgabetyp mit `|`
}
```

Hierbei handelt es sich nicht um ein generisches Typeset sondern einen Rückgabetyp mit `|`. Der ursprüngliche RegEx würde hier fälschlicherweise anschlagen, weswegen dieser robustere RegEx entwickelt wurde.

Neuer RegEx:
```
/type\s+?[a-zA-Z_]\w*?\s+?interface\s*?\{\s*?(?:~?[a-zA-Z_0-9\.\*]+(?:\s*?\|\s*?~?[a-zA-Z_0-9\.\*]+)+)\s*?\}/
```

**Suche nach Verwendung von Type Assertion:**  
Beispiel:
```go
price = first(product).(float32)
```
RegEx:
```
/\.\(.+?\)/
```

Die Ergebnisse der Untersuchungen laute dann (Stand 21.09.2025):

| Subset                                                   | #     |
| -------------------------------------------------------- | ----- |
| All indexed Go repos                                     | 47869 |
| 1) Repos declaring generic functions                     | 5459  |
| 2) Repos declaring type parameters                       | 3531  |
| 3) Repos declaring type sets                             | 1546  |
| Repos using type assertions                              | 31846 |
| => Repos being in 1), 2), or 3)                          | 5958  |
| => Repos being in 1), 2), or 3) or using type assertions | 32224 |

### Anmerkung zu RegEx und Motivation für einen GoParser
Eine wichtige Anmerkung zur Verwendung der oben aufgezeigten RegEx ist, dass diese mit der Hilfe von Sourcegraph den Text in den Dateien der jeweiligen Repositories untersuchen. Es wird dabei tatsächlich der Text und nicht nur der Code untersucht, wodurch die RegEx in einigen Fällen auch auf Repositories anschlugen, welche keine tatsächliche Verwendung von Generics beinhalteten und lediglich generischen **auskommentierten** Code beinhalteten. In diesen Fällen kann wohl kaum von der Verwendung von Generics in go gesprochen werden. Es handlet sich also offensichtlich um weitere false positives, welche sich nicht mit der Funktionalität von Sourcegraph beheben lassen. 
Aufgrund dessen entscheiden wir uns einen feingranulareren Ansatz zu verfolgen, welcher robuster gegenüber den zuvor erwähnten false positives ist. Bei diesem Ansatz handelt es sich um einen eigens entwickelten **GoParser** mit AST-Analyse.

## GoParser 
GoParser ist ein Tool zur Analyse von Go-Quellcode in GitHub-Repositorys mit Fokus auf die Verwendung von Generics in Go. Es verarbeitet Go-Dateien, um verschiedene generische und nicht-generische Konstrukte wie Funktionen, Methoden, Strukturen und Typdeklarationen zu zählen, und erstellt einen CSV-Bericht mit einer Zusammenfassung der Ergebnisse. Das Tool ist besonders nützlich, um die Akzeptanz und Implementierungsmuster von Generics in Go-Projekten zu untersuchen.

### Funktionsweise
GoParser lädt und analysiert Go-Quellcode aus angegebenen GitHub-Repositorys. Der Ablauf umfasst folgende Schritte:

1. Repository-Eingabe: Liest eine CSV-Datei ein, die GitHub-Repositorys im Format github.com/owner/repo enthält. Jede Zeile liefert den Eigentümer (owner) und den Repository-Namen (repo).
2. Download des Repositorys: Lädt das Repository als ZIP-Datei von GitHub herunter, indem es den Standard-Branch (z. B. main) verwendet. Hierfür wird die GitHub-API genutzt, wobei ein optionaler GitHub-Token für authentifizierte Zugriffe verwendet werden kann, um API-Ratenlimits zu erhöhen.
3. Entpacken und Filtern: Entpackt die ZIP-Datei und extrahiert alle Dateien mit der Endung .go.
4. Code-Analyse: Analysiert jede Go-Datei mithilfe des Go-parser-Pakets aus der Standardbibliothek. Dabei wird der Abstract Syntax Tree (AST) durchlaufen, um folgende Elemente zu zählen:

   - Funktionen: Gesamtzahl (FuncTotal) und generische Funktionen (FuncGeneric).
   - Methoden: Gesamtzahl (MethodTotal) und Methoden mit generischem Receiver (MethodWithGenericReceiver).
   - Strukturen: Gesamtzahl (StructTotal), generische Strukturen (StructGeneric) und generische Strukturen mit nicht-trivialen Constraints (StructGenericBound, d. h. Constraints ungleich any).
   - Typdeklarationen: Generische Typdeklarationen (GenericTypeDecl) und Type Sets in Interfaces (GenericTypeSet).


5. Aggregation: Summiert die Zähler für jedes Repository und aggregiert Statistiken über alle Repositorys hinweg, z. B. wie viele Repositorys generische Funktionen enthalten.
6. Ausgabe: Erzeugt eine CSV-Ausgabe für jedes Repository mit den gezählten Werten und eine abschließende Statistik über alle Repositorys.

Die Analyse erfolgt effizient durch die Verwendung des Go-AST, wodurch präzise Erkenntnisse über die Nutzung von Generics gewonnen werden können.

### Anleitung

Um GoParser zu verwenden muss Go in Version 1.18 oder höher installiert sein, da Generics erforderlich für das Tool sind. Zusätzlich wird ein Github-Token empfohlen, da man mithilfe dessen nicht in die API-Limits von Github läuft. Dieser Token muss dann in der main.go-Datei in die Variable **Token** eingefügt werden.

Zusätzlich wird eine Eingabe-csv benötigt, welche im Format *Match type,Repository,Repository external URL* vorliegen sollte. Diese Format entspricht den csvs, welche man bekommt, wenn man die Ergebnisse aus sourcegrapgh exportiert. Der Pfad zu dieser csv wird dann ebenfalls in der main.go-Datei in die Variable **csvPath** eingefügt.

Sind diese Vorraussetzungen erfüllt kann die Datei mit 
```
go run . > output.csv
```
ausgeführt werden. Die Ausgabe des Programmes wird dann automatisch in die Datei *output.csv* geschrieben.

Der GoParser kann dabei natürlich frei an Bedürfnisse und Anwendungsfälle angepasst werden. Die verschiedenen Dateien verfolgen dabei die folgenden Funktionen:
- **astAnalyzer.go**: Hier geschieht die AST-Analyse des Parsers.
- **csvUtil.go**: Liest die Zeilen der Eingabe csv Datei ein und formatiert sie für den Parser.
- **githubUtil.go**: Downloadet die Repositories aus der Eingabe csv Datei als .zip, entpackt diese und gibt die Dateien in einem String Array zurück.
- **main.go**: Orchestiert den Ablauf des Parsers und aggregiert die Werte.

### Ergebnisse

Im Rahmen dieser Projektarbeit wurde GoParser zweifach ausgeführt. Einerseits wurde es einmal auf 10.000 Repositories augeführt. Das Ergebnis davon liegt unter *output/finalOutput.csv*. Dies wurde mit einer veralteten Version des Tools gemacht, wodurch die Ergebnisse sich potentiell leicht von den endgültigen Werten entscheiden. Bei der Analyse dieses Outputs kam heraus, dass ca. 7% aller Repositories Generics verwenden. Im Gegensatz zu den ca. 8% der Sourcegraph-Analyse kommt GoParser damit auf einen Prozentpunkt weniger. Dies ist möglicherweise darauf zurückzuführen, dass die Sourcegraph-Analyse wie in obigem Abschnitt erklärt, anfälliger auf False Positives ist. Die Verwendung des GoParsers ermöglicht somit eine genauere Analyse.

Außerdem wurde das Tool auf zehn spezifisch ausgewählten, bekannteren Repositories ausgeführt um ein Gefühl für die Verwendung von Generics in großen Repositories zu bekommen. Die Ergebnisse davon sind in folgender Tabelle zu sehen:

| Repository                | FuncTotal | FuncGeneric | MethodTotal | MethodWithGenericReceiver | StructTotal | StructGeneric | StructGenericNonTrivialBound | TypeDecl | GenericTypeDecl | GenericTypeSet |
|--------------------------|-----------|-------------|-------------|---------------------------|-------------|---------------|------------------------------|----------|-----------------|----------------|
| prometheus/prometheus    | 3401      | 15          | 4141        | 10                        | 1055        | 5             | 4                            | 1397     | 6               | 3              |
| kubernetes/kubernetes    | 70454     | 364         | 94578       | 348                       | 21349       | 106           | 67                          | 29947    | 178             | 21             |
| golang/go                | 66800     | 1474        | 24446       | 567                       | 13723       | 466           | 126                          | 20399    | 826             | 223            |
| minio/minio              | 4739      | 21          | 5114        | 81                        | 1087        | 17            | 7                            | 1409     | 19              | 2              |
| moby/moby                | 34740     | 204         | 50107       | 407                       | 14953       | 117           | 47                           | 20292    | 151             | 5              |
| cockroachdb/cockroach    | 42355     | 171         | 49019       | 386                       | 11903       | 65            | 35                           | 15866    | 102             | 4              |
| etcd-io/etcd             | 4571      | 3           | 5420        | 11                        | 973         | 3             | 0                            | 1342     | 5               | 0              |
| hashicorp/terraform      | 6234      | 71          | 9790        | 91                        | 2090        | 29            | 17                           | 2750     | 49              | 0              |
| hashicorp/consul         | 9400      | 93          | 12684       | 123                       | 3581        | 44            | 34                           | 4463     | 68              | 0              |
| juju/juju                | 12192     | 149         | 81184       | 80                        | 20204       | 27            | 5                            | 23369    | 35              | 2              |


Die Ergebnisse stellen klar dar, dass Generics in Go vor allem in  großen, typintensiven Projekten wie golang/go und kubernetes/kubernetes weit verbreitet sind. Kleinere oder spezialisierte Projekte wie etcd-io/etcd setzen Generics gezielt, aber zurückhaltend ein. Generische Funktionen und Methoden kommen dabei unabhängig von der Repository-Größe deutlich häufiger vor als Generische Strukturen. Generell ist die Anzahl an Generischen Bausteinen bisher eher zurückhaltend. Gerade anhand des golang/go-Repositories kann man aber gut erkennen, dass Generics zumindest in großen Projekten regelmäßig verwendet werden.

Ein weiterer bemerkenswerter Punkt sind die hohe Gesamtanzahl an structs in kubernetes, golang und moby. Im Vergleich zu diesen ist die Anzahl von generischen structs bzw Funktionen und Methoden deutlich geringer. Generell ist das Verhältnis von structs zu Methoden und Funktionen etwa gleichbleibend. Ähnliche Zahlen liefert außerdem auch sourcegraph, wie [hier](https://sourcegraph.com/search?q=repo:%5Egithub%5C.com/kubernetes/kubernetes%24%40master+%5Cbtype%5Cs%2B(%5BA-Z%5D%5BA-Za-z0-9_%5D*)%5Cs%2Bstruct%5Cs*%5C%7B++count:50000&patternType=regexp&sm=0) zu sehen. Für ein genaueres Verständnis der Zusammenhänge dort, benötigt es noch weitere Forschung.


## Verwendung von Large Language Models zur Analyse von Generics in Go

Large Language Models (LLM) sind seit geraumer Zeit ein wichtiger Bestandteil der Informatik. Auch in dieser Arbeit wurden einige Versuche unternommen, die Fähigkeiten verschiedener Modelle zu verwenden, um die Verwendung von Generics in Go zu analysieren. Es wurden dabei die folgenden Untersuchungen mit Hilfe von ChatGPT, Gemini und Grok angestellt:

- **Analysieren von Limitationen von Genrics in Go**: </br>
  Herbei wurden die Modelle schlichtweg befragt, welche Limitationen bei der Verwendung von Generics in Go existieren. Die Ergebnisse der Modelle unterschieden sich dabei nicht nennenswert und beliefen sich auf oberflächige Bemängelung. Die Ergebnisse der Modelle waren allerdings nicht nutzbar für die Analyse dieser Arbiet, da sich die Modelle oft widersprachen und die Limitationen von Generics in Go nicht konsequent analysieren konnten.
- **Grobflächige Analyse**: </br>
  Zwar waren alle Modelle in der Lage einige Go Repositories zu benennen, welche Generics verwenden, jedoch handelte es sich hierbei stets um nur eine geringe Anzahl von Repositories. Ferner benannten die Modelle konsequent nur Repositories, welche meist den Begriff *Generics* entweder im Namen und/oder der Beschreibung des Repositories enthielten. Repositories, welche also nicht direkt auf die Verwendung oder Aufarbeitung von Generics in Go spezialisiert waren, wurden nicht aufgezeigt. Auch diese Ergebnisse der Modelle wurden nicht in dieser Arbeit verwendet, da sie der grobflächigen Analyse mit Sourcegraph in allen Hinsichten unterlegen waren.
- **Feingranulare Analyse**: </br>
   Die Modelle wurden auch verwendet, um die Verwendung von Generics in konkreten Dateien zu analysieren. Hierbei wurde das Aufkommen von generischen Funktionen, generischen Typdeklarationen und die Verwendung von generischen TypeSets berücksichtigt. Zwar waren die Modelle in der Lage diese generischen Bausteine robust zu erkennen, allerdings war es nicht möglich große Repositories mit Hilfe der Modelle zu untersuchen, da die Modelle zeitlich und ressourcen-technisch nicht in der Lage waren mehrere Hunderttausende Dateien zu analysiern. Die Ergebnisse dieser Untersuchung wurden daher in dieser Arbeit nicht verwendet, da die Modelle der feingranularen Analyse des **GoParsers** klar unterlegen waren.

## Fazit

Die Einführung von Generics in Go mit Version 1.18 markiert einen bedeutenden Schritt in der Entwicklung der Sprache, indem sie die Wiederverwendbarkeit und Flexibilität des Codes erhöht, ohne die charakteristische Einfachheit und Performanz von Go zu opfern. Generics ermöglichen es Entwicklern, typsicheren, wiederverwendbaren Code zu schreiben, der mit einer Vielzahl von Datentypen funktioniert, wie die Beispiele für generische Funktionen, Strukturen und Type Sets in dieser Arbeit zeigen. Die Verwendung von Type Parameters, Constraints und Type Inference erleichtert die Entwicklung generischer Programme, während die Implementierung durch Monomorphisierung eine hohe Laufzeitperformance gewährleistet.
Dennoch zeigen die analysierten Limitationen, wie fehlender Support für polymorphe Rekursion, Code-Bloat und eingeschränkte Expressivität von Constraints, dass Go-Generics bewusst pragmatisch und einfach gehalten wurden. Dies spiegelt die Design-Philosophie der Sprache wider, die Komplexität zu minimieren, führt jedoch zu Einschränkungen im Vergleich zu Sprachen wie Rust oder Haskell. Die Analyse mit Sourcegraph und dem speziell entwickelten GoParser zeigt, dass Generics in Go vor allem in großen, typintensiven Projekten wie golang/go und kubernetes/kubernetes weit verbreitet sind, während kleinere Projekte sie zurückhaltender einsetzen. Der GoParser erweist sich dabei als robustes Werkzeug, das durch die Nutzung des Go-AST präzisere Ergebnisse liefert als reguläre Ausdrücke, insbesondere durch die Vermeidung von False Positives wie auskommentiertem Code.
Die Untersuchung der Nutzung von Large Language Models (LLMs) zur Analyse von Generics zeigt, dass diese derzeit in ihrer Fähigkeit begrenzt sind, tiefgehende oder großflächige Analysen durchzuführen. Während sie grundlegende Erkenntnisse liefern können, sind sie spezialisierten Tools wie dem GoParser in Bezug auf Genauigkeit und Skalierbarkeit unterlegen.
Zusammenfassend bieten Generics in Go eine wertvolle Ergänzung für die Entwicklung flexibler und wartbarer Software, insbesondere in großen Projekten. Die bewussten Einschränkungen und die fortschreitende Optimierung durch das Go-Team deuten darauf hin, dass die Balance zwischen Einfachheit und Funktionalität weiter verfeinert wird. Der GoParser stellt ein vielversprechendes Werkzeug dar, um die Akzeptanz und Implementierungsmuster von Generics in der Go-Community weiter zu untersuchen.

## Ausblick

Für zukünftige Untersuchungen zu Generics in Go bieten sich mehrere Ansätze an, um deren Verwendung, Akzeptanz und Auswirkungen umfassender zu beleuchten. Ein zentraler Schritt wäre eine vertiefte semantische Analyse, die nicht nur syntaktische Vorkommen erfasst, sondern auch den kontextuellen Einsatz sowie Effizienz, Integration in bestehende Projekte und die Auswirkungen auf Codequalität, Performanz und Wartbarkeit untersucht. Darüber hinaus fehlt bislang eine repräsentative Befragung von Experten, die sowohl quantitative als auch qualitative Einblicke in die Wahrnehmung, Nutzung und Herausforderungen von Generics liefern könnte. In diesem Zusammenhang wurden bereits Fachleute aus Open-Source-Projekten wie Kubernetes und Golang kontaktiert. Im Zuge dessen wurden die Fachleute mit Fragen zur praktischen Verwendung und ihren Erfahrungen, zu Limitationen sowie zur Einschätzung der zukünftigen Entwicklung, um eine Einschätzung gebeten. Da bisher keine Rückmeldungen eingegangen sind, bleibt hier eine Lücke bestehen, die weitere Anstrengungen erfordert. Solche Erweiterungen könnten die Analyse um wertvolle qualitative Aspekte bereichern und zu fundierteren Empfehlungen sowie einer umfassenderen Bewertung der Rolle von Generics in Go beitragen. Besonders die Befragung von Fachleuten stellt eine wichtige Einsicht dar, um einen Überblick über die tatsächliche Einsetzung von Generics in weitverbreiteten und massiven Projekten zu beurteilen.



## Quellenverzeichnis
[1] Sulzmann, M., & Wehr, S. (2023). A Type-Directed, Dictionary-Passing Translation of Method Overloading and Structural Subtyping in Featherweight Generic Go. arXiv preprint arXiv:2209.08511. https://arxiv.org/abs/2209.08511 </br>
[2] https://go.dev/ref/spec#Type_parameter_declarations </br>
[3] https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#No-parameterized-methods </br>
[4] https://appliedgo.com/blog/a-tip-and-a-trick-when-working-with-generics </br>
[5] https://multithreaded.stitchfix.com/blog/2023/02/01/go-polymorphic-interfaces/ </br>
[6] https://www.dolthub.com/blog/2024-11-22-are-golang-generics-simple-or-incomplete-1/ </br>