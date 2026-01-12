/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/vcatafesta/chili/go

    Created:   dom 01 out 2023 09:00:00 -03
    Altered:   qui 05 out 2023 10:00:00 -03
    Updated:   seg 12 jan 2026 09:43:00 -04
    Version:   2.0.8

    Copyright (c) 2019-2026, Vilmar Catafesta <vcatafesta@gmail.com>
    Copyright (c) 2019-2026, ChiliLinux Team
    All rights reserved.

    Redistribution and use in source and binary forms, with or without
    modification, are permitted provided that the following conditions
    are met:
    1. Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
    2. Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.0.8-20260112"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

// --- CONFIGURAÇÃO DE CORES (IDENTIDADE CHILI) ---
var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	magenta = color.New(color.Bold, color.FgMagenta).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
)

// --- VARIÁVEIS GLOBAIS ---
var (
	inputFile     string
	engine        string
	jobs          int
	forceFlag     bool
	quietFlag     bool
	verboseFlag   bool
	languages     []string
	logger        *log.Logger
	cacheFile     string
	cacheData     map[string]map[string]string
	mu            sync.Mutex
	muConsole     sync.Mutex
	cacheHits     int
	netCalls      int
	isOnline      bool
	langsDone     int32
	langPositions map[string]int
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh-CN", "zh-TW",
}

var defaultLanguages = []string{"en", "es", "it", "de", "fr", "ru", "zh-CN", "zh-TW", "ja", "ko"}

// --- INICIALIZAÇÃO ---
func init() {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755)
	cacheFile = filepath.Join(cacheDir, "cache.json")
	langPositions = make(map[string]int)
}

func main() {
	checkDependencies()
	isOnline = checkInternet()

	// Definição de Argumentos
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor de tradução")
	pflag.IntVarP(&jobs, "jobs", "j", 8, "Traduções simultâneas")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar tradução")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo quieto")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, "Mostrar detalhes")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas")
	pflag.Usage = usage
	pflag.Parse()

	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	baseName := filepath.Base(inputFile)
	ext := strings.ToLower(filepath.Ext(baseName))
	isMarkdown := (ext == ".md" || ext == ".markdown")

	confLog()
	loadCache()
	defer saveCache() // Garante a gravação ao sair

	// Seleção de Idiomas
	targetLangs := defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}

	fmt.Printf("%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))

	// STEP 1: PREPARAÇÃO
	if isMarkdown {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Modo Markdown: Saída em doc/"))
		os.MkdirAll("doc", 0755)
	} else {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Extraindo strings para pot/ ..."))
		os.MkdirAll("pot", 0755)
		prepareGettext(inputFile, baseName)
	}

	// STEP 2: PROCESSAMENTO PARALELO
	fmt.Printf("%s %s %s %s\n", yellow("[STEP 2]"), white("Processando idiomas (Jobs:"), red(jobs), white(")"))
	
	for i, lang := range targetLangs {
		langPositions[lang] = len(targetLangs) - i
		fmt.Printf("   %s %-7s %s\n", blue("→"), cyan(lang), yellow("[Aguardando...]"))
	}

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs) // Controle de concorrência (Jobs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{} // Ocupa slot
			if isMarkdown {
				translateMarkdown(inputFile, l)
			} else {
				prepareMsginit(baseName, l)
				translateFile(baseName, l)
				writeMsgfmtToMo(baseName, l)
			}
			atomic.AddInt32(&langsDone, 1)
			<-sem // Libera slot
		}(lang)
	}
	wg.Wait()

	// RELATÓRIO FINAL
	fmt.Printf("\n%s %s | Cache: %d | Net: %d | Tempo: %v\n", 
		green("✔"), white("Concluído"), cacheHits, netCalls, time.Since(start).Round(time.Second))
}

// --- TRADUÇÃO DE MARKDOWN ---
func translateMarkdown(inputPath, lang string) {
	content, _ := os.ReadFile(inputPath)
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	total := len(lines)

	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outFile := filepath.Join("doc", fmt.Sprintf("%s-%s%s", base, lang, ext))

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Ignora blocos de código e linhas vazias
		if trimmed == "" || strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "`") {
			translatedLines = append(translatedLines, line)
		} else {
			translatedLines = append(translatedLines, callUniversalTranslator(line, lang))
		}
		if i%5 == 0 || i == total-1 {
			updateProgress(lang, i+1, total, "MD")
		}
	}
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	updateProgress(lang, total, total, "OK")
}

// --- TRADUÇÃO DE ARQUIVOS PO (GETTEXT) ---
func translateFile(baseName, lang string) {
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, lang))
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))

	file, _ := os.Open(poTmp)
	defer file.Close()
	output, _ := os.Create(poFinal)
	defer output.Close()

	var lines []string
	scannerCount := bufio.NewScanner(file)
	for scannerCount.Scan() { lines = append(lines, scannerCount.Text()) }
	
	totalMsgids := 0
	for _, l := range lines { if strings.HasPrefix(l, "msgid ") { totalMsgids++ } }

	current := 0
	var isMsgid bool
	var msgidLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "msgid ") {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
		} else if strings.HasPrefix(line, "msgstr ") && isMsgid {
			current++
			updateProgress(lang, current, totalMsgids, "PO")
			fullMsgid := strings.Join(msgidLines, " ")
			// Chama tradutor limpando aspas
			translated := callUniversalTranslator(strings.Trim(fullMsgid, `"`), lang)
			fmt.Fprintf(output, "msgid %s\n%s\n", strings.Join(msgidLines, "\n"), formatMsgstr(translated))
			isMsgid = false
		} else if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(output, line)
		}
	}
	updateProgress(lang, totalMsgids, totalMsgids, "OK")
}

// --- CORE: CHAMADA DO TRADUTOR COM PROTEÇÃO ---
func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" { return "" }
	normID := strings.ToLower(text)

	// Busca Cache
	mu.Lock()
	if val, ok := cacheData[lang][normID]; ok && !forceFlag {
		cacheHits++
		mu.Unlock()
		return val
	}
	mu.Unlock()

	if !isOnline { return text }

	// Proteção de Variáveis ($VAR, URLs, etc)
	protectedText, placeholders := protectVariables(text)
	netCalls++

	// Execução do 'trans'
	cmd := exec.Command("trans", "-e", engine, "-no-init", "-no-autocorrect", "-b", ":"+lang)
	cmd.Stdin = strings.NewReader(protectedText)
	out, err := cmd.Output()

	res := text
	if err == nil { res = strings.TrimSpace(string(out)) }
	
	// Restaura Variáveis
	res = restoreVariables(res, placeholders)

	// Salva no Cache em Memória
	mu.Lock()
	if _, ok := cacheData[lang]; !ok { cacheData[lang] = make(map[string]string) }
	cacheData[lang][normID] = res
	mu.Unlock()
	
	// Grava no disco periodicamente via defer no main ou aqui para segurança extra
	saveCache()

	return res
}

// --- PROTEÇÃO DE VARIÁVEIS E LINKS ---
func protectVariables(text string) (string, map[string]string) {
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z]|\[.*?\]\(.*?\)|<a\s+.*?>.*?</a>|https?://[^\s)\]]+)`)
	placeholders := make(map[string]string)
	matches := re.FindAllString(text, -1)
	protectedText := text
	for i, match := range matches {
		placeholder := fmt.Sprintf("CHILI_REF_%d_CHILI", i)
		placeholders[placeholder] = match
		protectedText = strings.Replace(protectedText, match, placeholder, 1)
	}
	return protectedText, placeholders
}

func restoreVariables(text string, placeholders map[string]string) string {
	for p, o := range placeholders { text = strings.Replace(text, p, o, -1) }
	return text
}

// --- UTILITÁRIOS E SISTEMA ---

func checkInternet() bool {
	_, err := net.DialTimeout("tcp", "google.com:80", 2*time.Second)
	return err == nil
}

func loadCache() {
	cacheData = make(map[string]map[string]string)
	file, err := os.ReadFile(cacheFile)
	if err == nil { json.Unmarshal(file, &cacheData) }
}

func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func formatMsgstr(text string) string {
	text = strings.ReplaceAll(text, `"`, `\"`)
	lines := strings.Split(text, "\n")
	if len(lines) == 1 { return fmt.Sprintf("msgstr \"%s\"", lines[0]) }
	res := "msgstr \"\"\n"
	for i, l := range lines {
		if i < len(lines)-1 { res += fmt.Sprintf("\"%s\\n\"\n", l) } else { res += fmt.Sprintf("\"%s\"", l) }
	}
	return res
}

func checkDependencies() {
	for _, bin := range []string{"xgettext", "msginit", "msgfmt", "trans"} {
		if _, err := exec.LookPath(bin); err != nil { log.Fatalf("Dependência faltando: %s", bin) }
	}
}

func confLog() {
	f, _ := os.OpenFile("/tmp/"+_APP_+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	logger = log.New(f, "", 0)
}

func prepareGettext(inputPath, baseName string) {
	potFile := filepath.Join("pot", baseName+".pot")
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, inputPath).Run()
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", potFile).Run()
}

func prepareMsginit(baseName, lang string) {
	potFile := filepath.Join("pot", baseName+".pot")
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, lang))
	os.Remove(poTmp)
	exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	exec.Command("sed", "-i", "s/charset=ASCII/charset=utf-8/g", poTmp).Run()
}

func writeMsgfmtToMo(baseName, lang string) {
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))
	moFile := filepath.Join(dir, baseName+".mo")
	exec.Command("msgfmt", poFinal, "-o", moFile).Run()
}

func usage() {
	fmt.Printf("\n%s %s\nUso: %s -i <arquivo> [opções]\n", cyan(_APP_), white(_VERSION_), yellow(_APP_))
	pflag.PrintDefaults()
}

// --- INTERFACE DE PROGRESSO DINÂMICO ---
func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag || verboseFlag { return }
	muConsole.Lock()
	defer muConsole.Unlock()

	pos := langPositions[lang]
	percent := (current * 100) / total

	// Barra de 50 caracteres (Estilo Chili NPM)
	width := 50
	filled := (percent * width) / 100
	if percent > 0 && filled == 0 { filled = 1 }
	if filled > width { filled = width }

	// Renderização visual: ░ (ASCII 176) em azul
	barFilled := strings.Repeat("░", filled)
	barEmpty  := strings.Repeat(" ", width-filled)
	cBar := blue(barFilled) + barEmpty

	langField := fmt.Sprintf("%-6s", lang)
	status := cyan(suffix)
	if percent == 100 { status = green("OK") }

	// ANSI Escape: Sobe cursor, limpa linha, imprime barra e volta
	fmt.Printf("\033[%dA\r\033[K   %s %-8s [%s] %3d%% %-5s\033[%dB",
		pos, blue("→"), cyan(langField), cBar, percent, status, pos)
}
