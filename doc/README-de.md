`
# chili-übersetzer-go 🌶️

chili-translator-go ist ein universeller Wrapper für maschinelle Übersetzung, der in Go geschrieben wurde. Es wurde entwickelt, um Skripte (.sh, .py), Dokumentationsdateien (Markdown) und Datendateien (JSON) zu übersetzen und dabei die Integrität von Variablen, Links und technischer Syntax zu wahren.

Sein Hauptvorteil ist Smart Cache v2.1.9, der Netzwerkaufrufe drastisch reduziert und sich wiederholende Übersetzungen durch lokale Datenwiederverwendung beschleunigt.

## ✨ Funktionen

* Multiformat: Unterstützt .sh, .py, .md, .json, .yaml.
* Syntaxerhaltung: Schützt automatisch Shell-Variablen ($VAR, ${VAR}), Markdown-Links und String-Platzhalter während des Übersetzungsprozesses.
* Parallele Übersetzung: Verarbeitet mehrere Sprachen gleichzeitig mithilfe von Goroutinen (einstellbar über -j).
* Persistenter Cache mit Zeitstempel: Speichert Übersetzungen lokal und verwaltet den Datenlebenszyklus, was eine intelligente Bereinigung ermöglicht.
* Progressive Schnittstelle: Echtzeitanzeige des Fortschritts jeder Sprache mit perfekter visueller Ausrichtung, unabhängig von der Sprachcodegröße (z. B. en vs zh-CN).

## 🚀 Installation

Stellen Sie sicher, dass Go installiert ist und die Systemabhängigkeiten (gettext, trans) vorhanden sind.
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-tradutor-go/src
go build -o chili-tradutor-go chili-tradutor-go-v2.1.9.go
sudo mv chili-tradutor-go /usr/local/bin/
```

## 🛠️ Nutzung

### Grundlegende Übersetzung
So übersetzen Sie eine Datei in Standardsprachen (EN, ES, IT, DE, FR, RU, ZH, JA, KO):

chili-translator-go -i meu_script.sh


### Angeben von Sprachen und Engine

cheli-treducer-go -und Tutorial.md


### Cache-Löschen
Entfernen Sie Cache-Einträge, die in den letzten 30 Tagen nicht verwendet wurden:

chili-translator-go --clean-cache


## ⚙️ Optionen (Flaggen)

| Flagge | Lang | Beschreibung |
| :--- | :--- | :--- |
| -i | --inputfile | Quelldatei zur Übersetzung. |
| -e | --engine | Übersetzungsmaschine: Google, Bing, Yandex (Standard: Google). |
| -s | --source | Quellsprache (z. B. pt, en) (Standard: automatisch). |
| -l | --Sprache | Liste der durch Komma oder alle Sprachen getrennten Sprachen. |
| -j | --jobs | Anzahl der Simultanübersetzungen (Standard: 8). |
| -f | --force | Erzwingt die Übersetzung unter Umgehung des lokalen Caches. |
| | --clean-cache | Entfernt veraltete Cache-Elemente (>30 Tage alt). |
| -q | --quiet | Silent-Modus (kein visueller Fortschritt). |
| -v | --verbose | Zeigt beim Laufen technische Details an. |
| -V | --version | Zeigt die aktuelle Version an. |

## 📁 Ausgabestruktur

* Skripte/POT: Erzeugt .po-Dateien in ./pot/ und .mo-Binärdateien in ./usr/share/locale/.
* Markdown: Erzeugt übersetzte Versionen in ./doc/ (z. B. README-en.md).
* JSON: Erzeugt übersetzte Versionen in ./translated/.

## 🛡️ Cache-Logik (v2.1.9)

Der Cache wird in ~/.cache/chili-tradutor-go/cache.json gespeichert.

* Automatische Migration: Beim Erkennen von Datensätzen aus früheren Versionen (v2.1.8) stempelt das Tool automatisch den aktuellen Zeitstempel in ältere Datensätze, um den Verlust historischer Daten zu vermeiden.
* Automatische Aktualisierung: Jedes Mal, wenn ein Element im Cache gefunden wird, wird sein Zeitstempel „Zuletzt verwendet“ aktualisiert, um es vor zukünftigen automatischen Löschungen zu schützen.
* Sicherheit: Beim Bereinigen über --clean-cache wird nur das entfernt, was tatsächlich nicht mehr verwendet wird, wodurch sichergestellt wird, dass Ihre Übersetzungswissensdatenbank gesund wächst.

---
Entwickelt von: Vilmar Catafesta <vcatafesta@gmail.com>
Copyright © 2023-2026 ChiliLinux Team
