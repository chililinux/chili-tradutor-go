`
#chili-traduttore-vai🌶️

chili-translator-go è un wrapper di traduzione automatica universale scritto in Go. È progettato per tradurre script (.sh, .py), file di documentazione (Markdown) e file di dati (JSON) mantenendo l'integrità di variabili, collegamenti e sintassi tecnica.

Il suo vantaggio principale è Smart Cache v2.1.9, che riduce drasticamente le chiamate di rete e accelera le traduzioni ripetitive attraverso il riutilizzo dei dati locali.

## ✨Caratteristiche

* Multiformato: supporta .sh, .py, .md, .json, .yaml.
* Conservazione della sintassi: protegge automaticamente le variabili della shell ($VAR, ${VAR}), i collegamenti Markdown e i segnaposto delle stringhe durante il processo di traduzione.
* Traduzione parallela: elabora più lingue contemporaneamente utilizzando Goroutines (sintonizzabile tramite -j).
* Cache persistente con timestamp: archivia le traduzioni localmente e gestisce il ciclo di vita dei dati, consentendo una pulizia intelligente.
* Interfaccia progressiva: visualizzazione in tempo reale dell'avanzamento di ciascuna lingua con un perfetto allineamento visivo, indipendentemente dalla dimensione del codice della lingua (ad esempio en vs zh-CN).

## 🚀 Installazione

Assicurati di avere Go installato e le dipendenze di sistema (gettext, trans).
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-translator-go/src
vai build -o chili-translator-go chili-translator-go-v2.1.9.go
sudo mv chili-translator-go /usr/local/bin/
```

## 🛠️ Utilizzo

### Traduzione di base
Per tradurre un file nelle lingue standard (EN, ES, IT, DE, FR, RU, ZH, JA, KO):

chili-translator-go -i meu_script.sh


### Specificare lingue e motore

cheli-treducer-go -e tutorial.md


### Cancellazione della cache
Rimuovi le voci della cache che non sono state utilizzate negli ultimi 30 giorni:

chili-translator-go --clean-cache


## ⚙️ Opzioni (Flag)

| Bandiera | Lungo | Descrizione |
| :--- | :--- | :--- |
| -io | --fileinput | File sorgente per la traduzione. |
| -e | --motore | Motore di traduzione: Google, Bing, Yandex (predefinito: Google). |
| -s | --fonte | Lingua di origine (es: pt, en) (impostazione predefinita: auto). |
| -l | --lingua | Elenco delle lingue separate da virgola o tutte. |
| -j | --lavori | Numero di traduzioni simultanee (default: 8). |
| -f | --forza | Forza la traduzione bypassando la cache locale. |
| | --clean-cache | Rimuove gli elementi della cache obsoleti (più vecchi di 30 giorni). |
| -q | --tranquillo | Modalità silenziosa (nessun progresso visivo). |
| -v | --verboso | Visualizza i dettagli tecnici durante la corsa. |
| -V | --versione | Visualizza la versione corrente. |

## 📁 Struttura dell'output

* Script/POT: genera file .po in ./pot/ e file binari .mo in ./usr/share/locale/.
* Markdown: genera versioni tradotte in ./doc/ (es: README-en.md).
* JSON: genera versioni tradotte in ./translated/.

## 🛡️ Logica cache (v2.1.9)

La cache è archiviata in ~/.cache/chili-tradutor-go/cache.json.

* Migrazione automatica: quando rileva record da versioni precedenti (v2.1.8), lo strumento applica automaticamente il timestamp corrente sui record legacy per evitare la perdita di dati storici.
* Aggiornamento automatico: ogni volta che un elemento viene trovato nella cache, il timestamp "Ultimo utilizzo" viene aggiornato, proteggendolo da future pulizie automatiche.
* Sicurezza: la pulizia tramite --clean-cache rimuove solo ciò che è effettivamente fuori uso, garantendo che la base di conoscenza della traduzione cresca in modo sano.


Sviluppato da: Vilmar Catafesta <vcatafesta@gmail.com>
Copyright © 2023-2026 ChiliLinux Team
