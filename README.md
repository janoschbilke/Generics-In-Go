# Generics in Go

Dieses Projekt beschäftigt sich mit der Implementierung und Anwendung von Generics in der Programmiersprache Go. Ziel ist es, die Vorteile, Herausforderungen und typischen Anwendungsfälle von Generics zu analysieren. Dabei werden sowohl theoretische Grundlagen als auch praktische Beispiele betrachtet. Ausführliche Informationen und weiterführende Dokumentationen sind im Docs-Ordner (`./docs/*`) zu finden.

## Setup

Für die Ausführung des Programms werden vorab zwei Werte erwartet: Ein GitHub Personal Access Token und der Input-Pfad zur CSV-Datei.
Die Bereitstellung dieser Werte kann über verschiedene Herangehensweisen erfolgen.

### Direkter Export als env-Variable

Man kann sie direkt als env-Variablen exportieren.
Dazu im Terminal folgende Befehle ausführen:

```bash
export GITHUB_TOKEN=ghp_...
export CSV_PATH=./Pfad/Zur/CSV/Datei
```

Default Wert für den CSV_PATH ist `../input/alleSourcegraph.csv`.

### Bereitstellung durch eine Secret-Datei

Die Werte können auch in einer *.env-Datei hinterlegt werden.
Standardmäßig wird die Datei `./secrets.env` im Project verwendet.

Es kann alternativ ein eigener Pfad zur Datei angegeben werden.
Dies geht auf zwei Arten:
Entweder kann vor dem Ausführen der Pfad zur Datei über eine Environment-Variable gesetzt werden:

```bash
export GOPARSER_SECRETS_PATH=./Pfad/zu/der/Datei
```

oder über die launch-Konfiguration von VSC konfiguriert werden.
In VSC sieht die Launch-Konfiguration dann beispielsweise so aus:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/GoParser/",
            "envFile": "${workspaceFolder}/secrets.env"
        }
    ]
}
```

In der Datei selbst muss das Token und der Pfad zu der CSV-Datei eingefügt werden.
Dabei gilt folgendes Format:

```env
GITHUB_TOKEN=ghp_...
CSV_PATH=../input/alleSourcegraph.csv
```

Anschließend kann das Programm wie gewohnt ausgeführt werden.
Bei Fehlern bitte den Output des Programms selbst betrachten.

## Local Development Setup

Das Programm unterstützt neben der Analyse von GitHub-Repositories auch die Analyse von **lokalen Go-Projekten**.
Das ist nützlich für das Testen spezifischer Verhaltensmuster oder die Entwicklung neuer Features.

### Aktivierung des lokalen Modus

Um ein lokales Projekt zu analysieren, muss die Environment-Variable `LOCAL_PROJECT_PATH` gesetzt werden. Dies kann, analog zum generellen Projekt Setup, auf verschiedene Arten erfolgen:

#### Option 1: Direkt als Environment-Variable

```bash
export LOCAL_PROJECT_PATH=/path/to/your/local/project
cd GoParser
go run .
```

#### Option 2: In der secrets.env Datei

Füge die folgende Zeile zu deiner `secrets.env` Datei hinzu:

```env
GITHUB_TOKEN=ghp_...
CSV_PATH=../input/alleSourcegraph.csv
LOCAL_PROJECT_PATH=/absolute/path/to/LocalTestProject
```

**Hinweis:** Der Pfad sollte absolut sein, z.B.:

- macOS/Linux: `/Users/username/project/LocalTestProject`
- Windows: `C:/Users/username/project/LocalTestProject`

### Verhalten im lokalen Modus

Wenn `LOCAL_PROJECT_PATH` gesetzt ist:

- Das Programm **ignoriert** die CSV-Datei und GitHub-Repositories
- Es durchsucht **rekursiv** alle `.go`-Dateien im angegebenen Verzeichnis
- Verzeichnisse wie `vendor`, `.git`, `node_modules` werden automatisch übersprungen
- Die Analyse erfolgt mit den gleichen Metriken wie bei GitHub-Repositories

### Test-Projekt

Im Repository ist ein `LocalTestProject` enthalten, das verschiedene Generic-Patterns demonstriert:

- Generic Functions mit Type Constraints
- Generic Structs mit trivialen (`any`) und nicht-trivialen Constraints
- Methods mit Generic Receivers
- Type Sets (Union Types)
- Normale (nicht-generische) Funktionen und Structs zum Vergleich

### Wechsel zwischen Modi

Um vom lokalen Modus zurück zum GitHub-Modus zu wechseln, entferne oder kommentiere die `LOCAL_PROJECT_PATH` Variable:

```env
# LOCAL_PROJECT_PATH=/path/to/project
GITHUB_TOKEN=ghp_...
CSV_PATH=../input/alleSourcegraph.csv
```
