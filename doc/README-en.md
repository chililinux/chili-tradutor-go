`
# chili-translator-go 🌶️

chili-translator-go is a universal machine translation wrapper written in Go. It is designed to translate scripts (.sh, .py), documentation files (Markdown), and data files (JSON) while maintaining the integrity of variables, links, and technical syntax.

Its main advantage is Smart Cache v2.1.9, which drastically reduces network calls and speeds up repetitive translations through local data reuse.

## ✨ Features

* Multiformat: Supports .sh, .py, .md, .json, .yaml.
* Syntax Preservation: Automatically protects shell variables ($VAR, ${VAR}), Markdown links and string placeholders during the translation process.
* Parallel Translation: Processes multiple languages simultaneously using Goroutines (tunable via -j).
* Persistent Cache with Timestamp: Stores translations locally and manages the data lifecycle, enabling intelligent cleaning.
* Progressive Interface: Real-time display of the progress of each language with perfect visual alignment, regardless of the language code size (e.g. en vs zh-CN).

## 🚀 Installation

Make sure you have Go installed and the system dependencies (gettext, trans).
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-tradutor-go/src
go build -o chili-tradutor-go chili-tradutor-go-v2.1.9.go
sudo mv chili-tradutor-go /usr/local/bin/
```

## 🛠️ Usage

### Basic Translation
To translate a file into standard languages (EN, ES, IT, DE, FR, RU, ZH, JA, KO):

chili-translator-go -i meu_script.sh


### Specifying Languages and Engine

cheli-treducer-go -and tutorial.md


### Cache Clearing
Remove cache entries that have not been used in the last 30 days:

chili-translator-go --clean-cache


## ⚙️ Options (Flags)

| Flag | Long | Description |
| :--- | :--- | :--- |
| -i | --inputfile | Source file for translation. |
| -e | --engine | Translation engine: Google, Bing, Yandex (default: Google). |
| -s | --source | Source language (ex: pt, en) (default: auto). |
| -l | --language | List of languages separated by comma or all. |
| -j | --jobs | Number of simultaneous translations (default: 8). |
| -f | --force | Forces translation by bypassing the local cache. |
| | --clean-cache | Removes stale cache items (>30 days old). |
| -q | --quiet | Silent mode (no visual progress). |
| -v | --verbose | Displays technical details while running. |
| -V | --version | Displays the current version. |

## 📁 Output Structure

* Scripts/POT: Generates .po files in ./pot/ and .mo binary files in ./usr/share/locale/.
* Markdown: Generates translated versions in ./doc/ (ex: README-en.md).
* JSON: Generates translated versions in ./translated/.

## 🛡️ Cache Logic (v2.1.9)

The cache is stored in ~/.cache/chili-tradutor-go/cache.json.

* Automatic Migration: When detecting records from previous versions (v2.1.8), the tool automatically stamps the current timestamp on legacy records to avoid loss of historical data.
* Auto-Update: Each time an item is found in the cache, its "Last Used" timestamp is updated, protecting it from future automatic clears.
* Security: Cleaning via --clean-cache only removes what is actually out of use, ensuring that your translation knowledge base grows healthily.

---
Developed by: Vilmar Catafesta <vcatafesta@gmail.com>
Copyright © 2023-2026 ChiliLinux Team
