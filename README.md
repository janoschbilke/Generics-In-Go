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

Die Werte können auch in einer *.env-Datei hinterlegt werden..
Um die Datei bereitzustellen, muss vor dem Ausführen der Pfad zur Datei entweder über

```bash
export GOPARSER_SECRETS_PATH=./Pfad/zu/der/Datei
```

oder über die launch-Konfiguration von VSC oder GoLand konfiguriert.
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
