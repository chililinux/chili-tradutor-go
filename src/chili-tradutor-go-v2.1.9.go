/*
	chili-tradutor-go
	Wrapper universal de tradução automática com cache inteligente

	Site:      https://chililinux.com
	GitHub:    https://github.com/vcatafesta/chili/go

	Created:   dom 01 out 2023 09:00:00 -03
	Altered:   qui 05 out 2023 10:00:00 -03
	Updated:   seg 12 jan 2026 16:05:00 -04
	Version:   2.1.9

	Copyright (c) 2019-2026, Vilmar Catafesta <vcatafesta@gmail.com>
	Copyright (c) 2019-2026, ChiliLinux Team
	All rights reserved.
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

// Estrutura para o Cache com Timestamp (v2.1.9)
type CacheEntry struct {
	Value    string    `json:"v"`
	LastUsed time.Time `json:"t"`
}

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.1.9-20260112"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

// --- CONFIGURAÇÃO DE CORES ---
var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	magenta = color.New(color.Bold, color.FgMagenta).SprintFunc()
)

// --- VARIÁVEIS GLOBAIS ---
var (
	inputFile       string
	engine          string
	sourceLang      string
	jobs            int
	forceFlag       bool
	quietFlag       bool
	verboseFlag     bool
	versionFlag     bool
	cleanCacheFlag  bool
	languages       []string
	logger          *log.Logger
	cacheFile       string
	cacheData       map[string]map[string]CacheEntry
	mu              sync.Mutex
	muConsole       sync.Mutex
	cacheHits       int
	netCalls        int
	failedCalls     int32
	isOnline        bool
	langsDone       int32
	langPositions   map[string]int
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh-CN", "zh-TW",
}

var defaultLanguages = []string{"en", "es", "it", "de", "fr", "ru", "zh-CN", "zh-TW", "ja", "ko"}

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

	pflag.Usage = usage
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "")
	pflag.StringVarP(&engine, "engine", "e", "google", "")
	pflag.StringVarP(&sourceLang, "source", "s", "auto", "")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "")
	pflag.IntVarP(&jobs, "jobs", "j", 8, "")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "")
	pflag.BoolVar(&cleanCacheFlag, "clean-cache", false, "")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, "")
	pflag.BoolVarP(&versionFlag, "version", "V", false, "")
	pflag.Parse()

	if versionFlag {
		showVersion()
		os.Exit(0)
	}

	loadCache()

	if cleanCacheFlag {
		doCleanCache()
		saveCache()
		os.Exit(0)
	}

	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	defer saveCache()

	baseName := filepath.Base(inputFile)
	ext := strings.ToLower(filepath.Ext(baseName))

	targetLangs := defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}

	fmt.Printf("%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))

	fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Analisando formato do arquivo e preparando ambiente..."))
	if ext == ".md" || ext == ".markdown" {
		os.MkdirAll("doc", 0755)
	} else if ext == ".json" || ext == ".yaml" || ext == ".yml" {
		os.MkdirAll("translated", 0755)
	} else {
		os.MkdirAll("pot", 0755)
		if ext == ".sh" || ext == ".py" {
			prepareGettext(inputFile, baseName)
		}
	}

	fmt.Printf("%s %s\n", yellow("[STEP 2]"), white("Configurações de tradução:"))
	fmt.Printf("    %s %-8s: %s\n", blue("→"), white("Origem"), magenta(sourceLang))
	fmt.Printf("    %s %-8s: %s\n", blue("→"), white("Idiomas"), cyan(strings.Join(targetLangs, ", ")))
	fmt.Printf("    %s %-8s: %s\n", blue("→"), white("Motor"), green(engine))
	fmt.Printf("    %s %-8s: %s\n", blue("→"), white("Jobs"), red(jobs))

	netStatus := green("Online (Internet OK)")
	if !isOnline {
		netStatus = red("Offline (Apenas Cache)")
	}
	fmt.Printf("    %s %-8s: %s\n", blue("→"), white("Rede"), netStatus)
	fmt.Println()

	totalLangs := len(targetLangs)
	for i, lang := range targetLangs {
		langPositions[lang] = totalLangs - i
		langStr := fmt.Sprintf("%-7s", lang)
		fmt.Printf("    %s %s %s\n", blue("→"), cyan(langStr), yellow("[Aguardando...]"))
	}

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}

			if ext == ".md" || ext == ".markdown" {
				translateMarkdown(inputFile, l)
			} else if ext == ".json" {
				translateJSON(inputFile, l)
			} else {
				prepareMsginit(baseName, l)
				translateFile(baseName, l)
				writeMsgfmtToMo(baseName, l)
			}

			atomic.AddInt32(&langsDone, 1)
			muConsole.Lock()
			fmt.Printf("\r    %s %s / %s idiomas concluídos...", yellow("[STATUS]"), green(langsDone), green(totalLangs))
			muConsole.Unlock()
			<-sem
		}(lang)
	}
	wg.Wait()

	showQuickStats(start)
	showFinalSummary(start)
}

// --- UI STATS & ALINHAMENTO ---

func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag {
		return
	}
	muConsole.Lock()
	defer muConsole.Unlock()

	pos := langPositions[lang]
	if pos == 0 {
		return
	}

	percent := (current * 100) / total
	width := 50
	filled := (percent * width) / 100
	bar := blue(strings.Repeat("░", filled)) + strings.Repeat(" ", width-filled)

	langStr := fmt.Sprintf("%-7s", lang)
	// Padding horizontal ajustado com 4 espaços antes da seta
	fmt.Printf("\033[%dA\r\033[K    %s %s [%s] %3d%% %-5s\033[%dB",
		pos, blue("→"), cyan(langStr), bar, percent, cyan(suffix), pos)
}

func showQuickStats(start time.Time) {
	total := cacheHits + netCalls
	pCache, pNet := 0.0, 0.0
	if total > 0 {
		pCache = (float64(cacheHits) / float64(total)) * 100
		pNet = (float64(netCalls) / float64(total)) * 100
	}
	fmt.Printf("\n\n%s %s em %v | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n",
		green("✔"), white("Concluído"), time.Since(start).Round(time.Second),
		blue("Cache:"), cacheHits, pCache, yellow("Net:"), netCalls, pNet, white("Total:"), total)
}

func showFinalSummary(start time.Time) {
	fmt.Printf("%s\n %s\n", white(strings.Repeat("-", 60)), yellow("RESUMO EXECUTIVO:"))
	fmt.Printf("    %s %-15s: %v\n", blue("→"), "Tempo Total", time.Since(start).Round(time.Second))
	fmt.Printf("    %s %-15s: %d\n", blue("→"), "Cache Hits", cacheHits)
	fmt.Printf("    %s %-15s: %d\n", blue("→"), "Chamadas Rede", netCalls)
	if atomic.LoadInt32(&failedCalls) > 0 {
		fmt.Printf("    %s %-15s: %s\n", red("→"), "Falhas", red(atomic.LoadInt32(&failedCalls)))
	}
	fmt.Printf("%s\n\n", white(strings.Repeat("-", 60)))
}

// --- CORE: TRADUTOR ---

func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	normID := strings.ToLower(text)

	mu.Lock()
	if _, ok := cacheData[lang]; !ok {
		cacheData[lang] = make(map[string]CacheEntry)
	}
	if entry, exists := cacheData[lang][normID]; exists && !forceFlag {
		entry.LastUsed = time.Now()
		cacheData[lang][normID] = entry
		cacheHits++
		mu.Unlock()
		return entry.Value
	}
	mu.Unlock()

	if !isOnline {
		return text
	}
	protectedText, placeholders := protectVariables(text)

	var res string
	var err error
	for i := 0; i < 3; i++ {
		cmd := exec.Command("trans", "-e", engine, "-s", sourceLang, "-no-init", "-no-autocorrect", "-b", ":"+lang)
		cmd.Stdin = strings.NewReader(protectedText)
		out, errCmd := cmd.Output()
		if errCmd == nil {
			res = restoreVariables(strings.TrimSpace(string(out)), placeholders)
			err = nil
			break
		}
		err = errCmd
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	if err != nil {
		atomic.AddInt32(&failedCalls, 1)
		return text
	}

	netCalls++
	mu.Lock()
	cacheData[lang][normID] = CacheEntry{Value: res, LastUsed: time.Now()}
	mu.Unlock()
	return res
}

// --- PROCESSADORES ---

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

func translateFile(baseName, lang string) {
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, lang))
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))
	file, _ := os.Open(poTmp)
	defer file.Close()
	output, _ := os.Create(poFinal)
	defer output.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	totalMsgids := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "msgid ") {
			totalMsgids++
		}
	}

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
			translated := callUniversalTranslator(strings.Trim(strings.Join(msgidLines, " "), `"`), lang)
			fmt.Fprintf(output, "msgid %s\nmsgstr \"%s\"\n", strings.Join(msgidLines, "\n"), translated)
			isMsgid = false
		} else if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(output, line)
		}
	}
	updateProgress(lang, totalMsgids, totalMsgids, "OK")
}

func translateJSON(path, lang string) {
	data, _ := os.ReadFile(path)
	var obj map[string]interface{}
	json.Unmarshal(data, &obj)
	translateMap(obj, lang)
	out, _ := json.MarshalIndent(obj, "", "  ")
	outFile := filepath.Join("translated", fmt.Sprintf("%s-%s.json", strings.TrimSuffix(filepath.Base(path), ".json"), lang))
	os.WriteFile(outFile, out, 0644)
	updateProgress(lang, 100, 100, "JSON")
}

func translateMap(m map[string]interface{}, lang string) {
	for k, v := range m {
		if val, ok := v.(string); ok {
			m[k] = callUniversalTranslator(val, lang)
		} else if valMap, ok := v.(map[string]interface{}); ok {
			translateMap(valMap, lang)
		}
	}
}

// --- AUXILIARES & PERSISTÊNCIA ---

func loadCache() {
	cacheData = make(map[string]map[string]CacheEntry)
	file, err := os.ReadFile(cacheFile)
	if err == nil {
		json.Unmarshal(file, &cacheData)

		now := time.Now()
		modified := false
		for lang := range cacheData {
			for id, entry := range cacheData[lang] {
				if entry.LastUsed.IsZero() {
					entry.LastUsed = now
					cacheData[lang][id] = entry
					modified = true
				}
			}
		}
		if modified {
			saveCache()
		}
	}
}

func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func doCleanCache() {
	limit := time.Now().AddDate(0, 0, -30)
	count := 0
	for l := range cacheData {
		for id, e := range cacheData[l] {
			if e.LastUsed.Before(limit) {
				delete(cacheData[l], id)
				count++
			}
		}
	}
	fmt.Printf("%s Removidos %d itens obsoletos do cache.\n", green("✔"), count)
}

func checkInternet() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func checkDependencies() {
	for _, bin := range []string{"xgettext", "msginit", "msgfmt", "trans"} {
		if _, err := exec.LookPath(bin); err != nil {
			log.Fatalf("Falta dependência: %s", bin)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "\n%s %s\n%s\n\n", cyan(_APP_), white(_VERSION_), white(_COPY_))
	fmt.Fprintf(os.Stderr, "%s: %s %s %s\n\n", yellow("Uso"), green(_APP_), yellow("-i"), green("<arquivo> [opções]"))
	fmt.Fprintf(os.Stderr, "%s:\n", yellow("Opções"))
	defLangs := strings.Join(defaultLanguages, ",")
	flags := []struct {
		short string
		long  string
		desc  string
	}{
		{"-i", "--inputfile", "Arquivo fonte (.sh, .py, .md, .json, .yaml)"},
		{"-e", "--engine", "Motor: google, bing, yandex (padrão: google)"},
		{"-s", "--source", "Idioma de origem (ex: pt, en) (padrão: auto)"},
		{"-l", "--language", fmt.Sprintf("Idiomas (ex: pt-BR,en) ou 'all' (padrão: %s)", defLangs)},
		{"-j", "--jobs", "Traduções simultâneas (padrão: 8)"},
		{"-f", "--force", "Força nova tradução (ignora cache)"},
		{"", "--clean-cache", "Remove entradas de cache não usadas há 30 dias (Protege registros legados)"},
		{"-q", "--quiet", "Modo silencioso"},
		{"-v", "--verbose", "Mostrar detalhes"},
		{"-V", "--version", "Mostra a versão do programa"},
	}
	for _, f := range flags {
		s := f.short
		if s == "" {
			s = "   "
		}
		fmt.Fprintf(os.Stderr, "  %s, %-12s %s\n", cyan(s), cyan(f.long), white(f.desc))
	}
}

func prepareGettext(input, base string) {
	pot := filepath.Join("pot", base+".pot")
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "-o", pot, input).Run()
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", pot).Run()
}

func prepareMsginit(base, lang string) {
	pot := filepath.Join("pot", base+".pot")
	po := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", base, lang))
	os.Remove(po)
	exec.Command("msginit", "--no-translator", "-l", lang, "-i", pot, "-o", po).Run()
}

func writeMsgfmtToMo(base, lang string) {
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	exec.Command("msgfmt", filepath.Join("pot", fmt.Sprintf("%s-%s.po", base, lang)), "-o", filepath.Join(dir, base+".mo")).Run()
}

func protectVariables(text string) (string, map[string]string) {
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z]|\[.*?\]\(.*?\)|https?://[^\s]+)`)
	placeholders := make(map[string]string)
	protected := text
	for i, match := range re.FindAllString(text, -1) {
		p := fmt.Sprintf("CHILI_REF_%d_CHILI", i)
		placeholders[p] = match
		protected = strings.Replace(protected, match, p, 1)
	}
	return protected, placeholders
}

func restoreVariables(text string, p map[string]string) string {
	for k, v := range p {
		text = strings.Replace(text, k, v, -1)
	}
	return text
}

func showVersion() { fmt.Printf("%s %s\n%s\n", cyan(_APP_), white(_VERSION_), white(_COPY_)) }
