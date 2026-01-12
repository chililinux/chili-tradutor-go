/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/vcatafesta/chili/go

    Created:   dom 01 out 2023 09:00:00 -03
    Altered:   qui 05 out 2023 10:00:00 -03
    Updated:   seg 12 jan 2026 11:15:00 -04
    Version:   2.1.5

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

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.1.5-20260112"
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
	inputFile     string
	engine        string
	jobs          int
	forceFlag     bool
	quietFlag     bool
	verboseFlag   bool
	versionFlag   bool
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

	// Definição de Argumentos com as descrições solicitadas
	pflag.Usage = usage
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo fonte para tradução (.sh, .py, .md)")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas (ex: pt-BR,en) ou 'all'")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor: google, bing, yandex")
	pflag.IntVarP(&jobs, "jobs", "j", 8, "Traduções simultâneas")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Força nova tradução (ignora cache)")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo silencioso")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, "Mostrar detalhes")
	pflag.BoolVarP(&versionFlag, "version", "V", false, "Mostra a versão do programa")
	
	pflag.Parse()

	if versionFlag {
		showVersion()
		os.Exit(0)
	}

	if inputFile == "" {
		pflag.Usage()
		os.Exit(1)
	}

	baseName := filepath.Base(inputFile)
	ext := strings.ToLower(filepath.Ext(baseName))
	isMarkdown := (ext == ".md" || ext == ".markdown")

	confLog()
	loadCache()
	defer saveCache()

	targetLangs := defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}

	fmt.Printf("%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))

	// STEP 1
	if isMarkdown {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Modo Markdown: Saída em doc/"))
		os.MkdirAll("doc", 0755)
	} else {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Extraindo strings para pot/ ..."))
		os.MkdirAll("pot", 0755)
		prepareGettext(inputFile, baseName)
	}

	// STEP 2
	fmt.Printf("%s %s\n", yellow("[STEP 2]"), white("Configurações de tradução:"))
	fmt.Printf("   %s %-8s: %s\n", blue("→"), white("Idiomas"), cyan(strings.Join(targetLangs, ", ")))
	fmt.Printf("   %s %-8s: %s\n", blue("→"), white("Motor"), green(engine))
	fmt.Printf("   %s %-8s: %s\n", blue("→"), white("Jobs"), red(jobs))
	
	netStatus := green("Online (Internet OK)")
	if !isOnline {
		netStatus = red("Offline (Usando apenas Cache Local)")
	}
	fmt.Printf("   %s %-8s: %s\n", blue("→"), white("Rede"), netStatus)
	fmt.Println()

	for i, lang := range targetLangs {
		langPositions[lang] = len(targetLangs) - i
		fmt.Printf("   %s %-7s %s\n", blue("→"), cyan(lang), yellow("[Aguardando...]"))
	}

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	totalLangs := len(targetLangs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			if isMarkdown {
				translateMarkdown(inputFile, l)
			} else {
				prepareMsginit(baseName, l)
				translateFile(baseName, l)
				writeMsgfmtToMo(baseName, l)
			}
			atomic.AddInt32(&langsDone, 1)
			
			muConsole.Lock()
			fmt.Printf("\r   %s %s / %s idiomas concluídos...", yellow("[STATUS]"), green(langsDone), green(totalLangs))
			muConsole.Unlock()
			
			<-sem
		}(lang)
	}
	wg.Wait()

	// CONCLUSÃO CONSOLIDADA
	totalCalls := cacheHits + netCalls
	pCache := 0.0
	pNet := 0.0
	if totalCalls > 0 {
		pCache = (float64(cacheHits) / float64(totalCalls)) * 100
		pNet = (float64(netCalls) / float64(totalCalls)) * 100
	}

	fmt.Printf("\n\n%s %s em %v | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n\n", 
		green("✔"), white("Concluído"), time.Since(start).Round(time.Second),
		blue("Cache:"), cacheHits, pCache,
		yellow("Net:"), netCalls, pNet,
		white("Total:"), totalCalls,
	)
}

// --- FUNÇÕES DE APOIO ---

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

func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" { return "" }
	normID := strings.ToLower(text)

	mu.Lock()
	if _, ok := cacheData[lang]; !ok { cacheData[lang] = make(map[string]string) }
	if val, ok := cacheData[lang][normID]; ok && !forceFlag {
		cacheHits++
		mu.Unlock()
		return val
	}
	mu.Unlock()

	if !isOnline { return text }

	protectedText, placeholders := protectVariables(text)
	netCalls++

	cmd := exec.Command("trans", "-e", engine, "-no-init", "-no-autocorrect", "-b", ":"+lang)
	cmd.Stdin = strings.NewReader(protectedText)
	out, err := cmd.Output()

	res := text
	if err == nil { res = strings.TrimSpace(string(out)) }
	res = restoreVariables(res, placeholders)

	mu.Lock()
	cacheData[lang][normID] = res
	mu.Unlock()
	saveCache()

	return res
}

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

func checkInternet() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil { return false }
	conn.Close()
	return true
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

func showVersion() {
	fmt.Printf("%s %s\n%s\n", cyan(_APP_), white(_VERSION_), white(_COPY_))
}

func usage() {
	fmt.Fprintf(os.Stderr, "\n%s %s\n", cyan(_APP_), white(_VERSION_))
	fmt.Fprintf(os.Stderr, "%s\n\n", white(_COPY_))
	fmt.Fprintf(os.Stderr, "%s: %s %s %s\n\n", yellow("Uso"), green(_APP_), yellow("-i"), green("<arquivo> [opções]"))
	fmt.Fprintf(os.Stderr, "%s:\n", yellow("Opções"))
	
	pflag.VisitAll(func(f *pflag.Flag) {
		short := ""
		if f.Shorthand != "" {
			short = fmt.Sprintf("-%s,", f.Shorthand)
		}
		fmt.Fprintf(os.Stderr, "  %s %-14s %s\n", 
			cyan(fmt.Sprintf("%4s", short)), 
			cyan("--"+f.Name), 
			white(f.Usage))
	})
	fmt.Fprintf(os.Stderr, "\n%s:\n", yellow("Exemplos"))
	fmt.Fprintf(os.Stderr, "  %s -i script.sh -l pt-BR,en\n", green(_APP_))
	fmt.Fprintf(os.Stderr, "  %s -i README.md -l all -e bing\n\n", green(_APP_))
}

func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag || verboseFlag { return }
	muConsole.Lock()
	defer muConsole.Unlock()

	pos := langPositions[lang]
	percent := 0
	if total > 0 { percent = (current * 100) / total }
	
	width := 50
	filled := (percent * width) / 100
	if percent > 0 && filled == 0 { filled = 1 }
	if filled > width { filled = width }

	barFilled := strings.Repeat("░", filled)
	barEmpty  := strings.Repeat(" ", width-filled)
	cBar := blue(barFilled) + barEmpty

	langField := fmt.Sprintf("%-6s", lang)
	status := cyan(suffix)
	if percent == 100 { status = green("OK") }

	fmt.Printf("\033[%dA\r\033[K   %s %-8s [%s] %3d%% %-5s\033[%dB",
		pos, blue("→"), cyan(langField), cBar, percent, status, pos)
}
