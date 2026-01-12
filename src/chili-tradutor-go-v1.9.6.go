/*
   chili-tradutor-go - v1.9.6
   Copyright (c) 2023-2026, Vilmar Catafesta <vcatafesta@gmail.com>
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
	_VERSION_ = "1.9.6-20260112"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	orange  = color.New(color.FgYellow).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	magenta = color.New(color.Bold, color.FgMagenta).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	boldred = color.New(color.Bold, color.FgRed).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
)

var (
	inputFile    string
	engine       string
	jobs         int
	forceFlag    bool
	quietFlag    bool
	verboseFlag  bool
	languages    []string
	logger       *log.Logger
	cacheFile    string
	cacheData    map[string]map[string]string
	mu           sync.Mutex
	cacheHits    int
	netCalls     int
	isOnline     bool
	langsDone    int32
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
}

func main() {
	checkDependencies()
	isOnline = checkInternet()

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

	info, err := os.Stat(inputFile)
	if err != nil || info.IsDir() {
		fmt.Printf("%s Arquivo inválido: %s\n", boldred("[!]"), inputFile)
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

	if isMarkdown {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Modo Markdown: Saída em doc/ com sufixo"))
		os.MkdirAll("doc", 0755)
	} else {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Extraindo strings para pot/ ..."))
		os.MkdirAll("pot", 0755)
		os.Remove(filepath.Join("pot", baseName+".pot"))
		prepareGettext(inputFile, baseName)
	}

	fmt.Printf("%s %s\n", yellow("[STEP 2]"), white("Processando idiomas"))
	fmt.Printf("%s %-20s %s\n", magenta("[LANGS]"), white("Alvos:"), cyan(strings.Join(targetLangs, " ")))
	fmt.Printf("%s %-21s %s\n", magenta("[CONN]"), white("Rede:"), connStatus())
	fmt.Printf("%s %-21s %s\n\n", magenta("[JOBS]"), white("Paralelos:"), red(jobs))

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if isMarkdown {
				translateMarkdown(inputFile, l)
			} else {
				prepareMsginit(baseName, l)
				translateFile(baseName, l)
				writeMsgfmtToMo(baseName, l)
				os.Remove(filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, l)))
			}
			atomic.AddInt32(&langsDone, 1)
		}(lang)
	}
	wg.Wait()

	if !quietFlag && !verboseFlag {
		fmt.Printf("\n   %s %s: %d/%d concluídos...\n", blue("→"), white("Progresso"), langsDone, len(targetLangs))
	}
	printStats(start)
}

func translateMarkdown(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil { return }

	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	total := len(lines)

	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	newFileName := fmt.Sprintf("%s-%s%s", base, lang, ext)
	outFile := filepath.Join("doc", newFileName)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "`") {
			translatedLines = append(translatedLines, line)
		} else {
			translated := callUniversalTranslator(line, lang)
			translatedLines = append(translatedLines, translated)
		}
		if !quietFlag && !verboseFlag && i%10 == 0 {
			fmt.Printf("\r   %s %-7s [%d/%d] MD ", blue("→"), cyan(lang), i, total)
		}
	}
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	if !quietFlag && !verboseFlag { fmt.Printf("\n") }
}

func translateFile(baseName, lang string) {
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, lang))
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))

	file, err := os.Open(poTmp)
	if err != nil { return }
	defer file.Close()

	output, err := os.Create(poFinal)
	if err != nil { return }

	countScanner := bufio.NewScanner(file)
	totalMsgids := 0
	for countScanner.Scan() {
		if strings.HasPrefix(countScanner.Text(), "msgid ") { totalMsgids++ }
	}
	file.Seek(0, 0)

	scanner := bufio.NewScanner(file)
	var isMsgid bool
	var msgidLines []string
	current := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid ") {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
		} else if strings.HasPrefix(line, "msgstr ") && isMsgid {
			current++
			if !quietFlag && !verboseFlag {
				fmt.Printf("\r   %s %-7s [%d/%d] %d%% ", blue("→"), cyan(lang), current, totalMsgids, (current*100)/totalMsgids)
			}
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
	output.Sync()
	output.Close()
	if !quietFlag && !verboseFlag { fmt.Printf("\n") }
}

func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" { return "" }
	normID := strings.ToLower(text)

	mu.Lock()
	if val, ok := cacheData[lang][normID]; ok && !forceFlag {
		cacheHits++
		mu.Unlock()
		return val
	}
	mu.Unlock()

	if !isOnline { return text }

	protectedText, placeholders := protectVariables(text)
	netCalls++

	// USANDO STDIN PARA O TRANS PARA EVITAR EXECUTAR LINKS
	cmd := exec.Command("trans", "-e", engine, "-no-init", "-no-autocorrect", "-b", ":"+lang)
	cmd.Stdin = strings.NewReader(protectedText)
	out, err := cmd.Output()

	res := text
	if err == nil {
		res = strings.TrimSpace(string(out))
	}
	res = restoreVariables(res, placeholders)

	mu.Lock()
	if _, ok := cacheData[lang]; !ok { cacheData[lang] = make(map[string]string) }
	cacheData[lang][normID] = res
	mu.Unlock()

	return res
}

func protectVariables(text string) (string, map[string]string) {
	// Regex para variáveis shell e Links Markdown [texto](url)
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z]|\[.*?\]\(.*?\))`)
	placeholders := make(map[string]string)
	matches := re.FindAllString(text, -1)
	protectedText := text
	for i, match := range matches {
		placeholder := fmt.Sprintf("CHILI%dCHILI", i)
		placeholders[placeholder] = match
		protectedText = strings.Replace(protectedText, match, placeholder, 1)
	}
	return protectedText, placeholders
}

func restoreVariables(text string, placeholders map[string]string) string {
	restoredText := text
	for placeholder, original := range placeholders {
		restoredText = strings.Replace(restoredText, placeholder, original, -1)
	}
	return restoredText
}

func connStatus() string {
	if isOnline { return green("ONLINE") }
	return red("OFFLINE (Modo Cache)")
}

func checkDependencies() {
	required := []string{"xgettext", "msginit", "msgfmt", "sed", "trans"}
	for _, bin := range required {
		if _, err := exec.LookPath(bin); err != nil { log.Fatalf("Dependência faltando: %s", bin) }
	}
}

func checkInternet() bool {
	timeout := 2 * time.Second
	_, err := net.DialTimeout("tcp", "google.com:80", timeout)
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

func confLog() {
	fileLog := "/tmp/" + _APP_ + ".log"
	logFile, _ := os.OpenFile(fileLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if quietFlag { logger = log.New(logFile, "", 0) } else { logger = log.New(io.MultiWriter(os.Stdout, logFile), "", 0) }
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

func printStats(start time.Time) {
	duration := time.Since(start)
	totalRequests := cacheHits + netCalls
	fmt.Printf("\n%s %s\n", green("✔"), white("Processo concluído!"))
	fmt.Printf("%s %s: %d | %s: %d | %s: %d | %s: %v\n",
		magenta("[STATS]"), white("Total"), totalRequests,
		white("Cache"), cacheHits, white("Net"), netCalls,
		white("Tempo"), duration.Round(time.Second))
}

func writeMsgfmtToMo(baseName, lang string) {
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))
	moFile := filepath.Join(dir, baseName+".mo")
	if _, err := os.Stat(poFinal); err == nil {
		exec.Command("msgfmt", poFinal, "-o", moFile).Run()
	}
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

func usage() {
	fmt.Printf("\n%s %s\n", yellow(">>"), white(_APP_))
	fmt.Printf("%s %s\n", yellow(">>"), white(_COPY_))
	fmt.Printf("%s %s\n\n", yellow(">>"), white(_VERSION_))
	fmt.Printf("%s\n", boldred("USO:"))
	fmt.Printf("  %s %s %s %s\n\n", cyan(_APP_), yellow("-i"), green("<arquivo>"), magenta("[opções]"))
	fmt.Printf("%s\n", boldred("OPÇÕES:"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-i"), yellow("--inputfile"), white("Arquivo de entrada (.sh, .md)"))
	fmt.Printf("  %s, %-15s %s %s\n", yellow("-l"), yellow("--language"), white("Idiomas ou"), magenta("'all'"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-j"), yellow("--jobs"), white("Paralelos (8)"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-f"), yellow("--force"), white("Forçar (Ignorar cache)"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-h"), yellow("--help"), white("Mostrar ajuda"))
}
