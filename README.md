`
# chili-tradutor-go 🌶️

O chili-tradutor-go é um wrapper universal de tradução automática escrito em Go. Ele foi projetado para traduzir scripts (.sh, .py), arquivos de documentação (Markdown) e arquivos de dados (JSON), mantendo a integridade de variáveis, links e sintaxe técnica.

Sua principal vantagem é o Cache Inteligente v2.1.9, que reduz drasticamente as chamadas de rede e acelera traduções repetitivas através da reutilização de dados locais.

## ✨ Funcionalidades

* Multiformato: Suporta .sh, .py, .md, .json, .yaml.
* Preservação de Sintaxe: Protege automaticamente variáveis de shell ($VAR, ${VAR}), links Markdown e placeholders de string durante o processo de tradução.
* Tradução Paralela: Processa múltiplos idiomas simultaneamente usando Goroutines (ajustável via -j).
* Cache Persistente com Timestamp: Armazena traduções localmente e gerencia o ciclo de vida dos dados, permitindo limpezas inteligentes.
* Interface Progressiva: Exibição em tempo real do progresso de cada idioma com alinhamento visual perfeito, independente do tamanho do código do idioma (ex: en vs zh-CN).

## 🚀 Instalação

Certifique-se de ter o Go instalado e as dependências de sistema (gettext, trans).
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-tradutor-go/src
go build -o chili-tradutor-go chili-tradutor-go-v2.1.9.go
sudo mv chili-tradutor-go /usr/local/bin/
```bash

## 🛠️ Uso

### Tradução Básica
Para traduzir um arquivo para os idiomas padrão (EN, ES, IT, DE, FR, RU, ZH, JA, KO):

chili-tradutor-go -i meu_script.sh


### Especificando Idiomas e Motor

chili-tradutor-go -i tutorial.md -l "pt-BR,en,es" -e bing


### Limpeza de Cache
Remova entradas de cache que não foram utilizadas nos últimos 30 dias:

chili-tradutor-go --clean-cache


## ⚙️ Opções (Flags)

| Flag | Longa | Descrição |
| :--- | :--- | :--- |
| -i | --inputfile | Arquivo fonte para tradução. |
| -e | --engine | Motor de tradução: google, bing, yandex (padrão: google). |
| -s | --source | Idioma de origem (ex: pt, en) (padrão: auto). |
| -l | --language | Lista de idiomas separados por vírgula ou all. |
| -j | --jobs | Número de traduções simultâneas (padrão: 8). |
| -f | --force | Força a tradução ignorando o cache local. |
| | --clean-cache | Remove itens de cache obsoletos (> 30 dias). |
| -q | --quiet | Modo silencioso (sem progresso visual). |
| -v | --verbose | Exibe detalhes técnicos durante a execução. |
| -V | --version | Exibe a versão atual. |

## 📁 Estrutura de Saída

* Scripts/POT: Gera arquivos .po em ./pot/ e arquivos binários .mo em ./usr/share/locale/.
* Markdown: Gera versões traduzidas em ./doc/ (ex: README-en.md).
* JSON: Gera versões traduzidas em ./translated/.

## 🛡️ Lógica de Cache (v2.1.9)

O cache é armazenado em ~/.cache/chili-tradutor-go/cache.json. 

* Migração Automática: Ao detectar registros de versões anteriores (v2.1.8), a ferramenta carimba automaticamente o timestamp atual nos registros legados para evitar a perda de dados históricos.
* Auto-Update: Cada vez que um item é encontrado no cache, seu timestamp de "Último Uso" é atualizado, protegendo-o de limpezas automáticas futuras.
* Segurança: A limpeza via --clean-cache só remove o que realmente está em desuso, garantindo que sua base de conhecimento de tradução cresça de forma saudável.

---
Desenvolvido por: Vilmar Catafesta <vcatafesta@gmail.com>  
Copyright © 2023-2026 ChiliLinux Team
