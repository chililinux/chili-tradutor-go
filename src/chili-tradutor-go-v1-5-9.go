/*
   chili-tradutor-go - v1.5.9
   Copyright (c) 2023-2026, Vilmar Catafesta <vcatafesta@gmail.com>
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "1.5.9-20260110"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

// Cores baseadas no seu script original
var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	orange  = color.New(color.FgYellow).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	black   = color.New(color.Bold, color.FgBlack).SprintFunc()
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
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh",
}

func init() {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755)
	cacheFile = filepath.Join(cacheDir, "cache.json")
}

func confLog() {
	fileLog := "/tmp/" + _APP_ + ".log"
	logFile, err := os.OpenFile(fileLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir log: %v", err)
	}
	if quietFlag {
		logger = log.New(logFile, "", 0)
	} else {
		logger = log.New(io.MultiWriter(os.Stdout, logFile), "", 0)
	}
}

func loadCache() {
	cacheData = make(map[string]map[string]string)
	file, err := os.ReadFile(cacheFile)
	if err == nil {
		json.Unmarshal(file, &cacheData)
	}
}

func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func main() {
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor de tradução")
	pflag.IntVarP(&jobs, "jobs", "j", 3, "Traduções simultâneas")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar tradução")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo quieto")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", true, "Modo verbose (mostra IDs)")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas")
	pflag.Parse()

	confLog()
	loadCache()
	defer saveCache()

	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	targetLangs := supportedLanguages
	if len(languages) > 0 {
		targetLangs = languages
	}

	// [STEP 1] - Extração
	prepareGettext(inputFile)

	// [STEP 2] - Cabeçalho de Processamento Individual
	logger.Printf("%s Traduzindo %s para %d idiomas com %d jobs usando [%s]...\n", 
		black("[STEP 2]"), magenta(inputFile), len(targetLangs), jobs, cyan(engine))

	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prepareMsginit(inputFile, l)
			translateFile(inputFile, l)
			writeMsgfmtToMo(inputFile, l)
			os.Remove(fmt.Sprintf("%s-temp-%s.po", inputFile, l))
		}(lang)
	}
	wg.Wait()
	logger.Printf("\n%s %s\n", green("✔"), white("Todas as traduções concluídas!"))
}

func translateFile(inputFile, lang string) {
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	poFinal := fmt.Sprintf("%s-%s.po", inputFile, lang)

	file, err := os.Open(poTmp)
	if err != nil { return }
	defer file.Close()

	output, _ := os.Create(poFinal)
	defer output.Close()

	scanner := bufio.NewScanner(file)
	var isMsgid bool
	var msgidLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid ") {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
		} else if strings.HasPrefix(line, "msgstr ") && isMsgid {
			fullMsgid := strings.Join(msgidLines, " ")
			msgstr := translateMessage(fullMsgid, lang)
			fmt.Fprintf(output, "msgid %s\nmsgstr %s\n\n", strings.Join(msgidLines, "\n"), msgstr)
			isMsgid = false
		} else if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(output, line)
		}
	}
}

func translateMessage(msgid, langCode string) string {
	msgid = strings.TrimSpace(msgid)
	cleanID := strings.Trim(msgid, `"`)
	if cleanID == "" { return `""` }

	// Lógica de Cache (Normalizada)
	normID := strings.ToLower(strings.TrimSpace(cleanID))
	mu.Lock()
	if _, ok := cacheData[langCode]; ok {
		if val, ok := cacheData[langCode][normID]; ok && !forceFlag {
			mu.Unlock()
			if verboseFlag {
				logger.Printf("%s (%s)\t%s [len=%-3d] %s %s => %s %s\n", 
					orange("[TRANS]"), cyan(langCode), yellow("ID:"), len(cleanID), white(cleanID), green("to"), blue(val), green("(CACHE)"))
			}
			return `"` + val + `"`
		}
	}
	mu.Unlock()

	// Tradução Real
	cmd := exec.Command("trans", "-e", engine, "-no-autocorrect", "-b", ":"+langCode, cleanID)
	out, err := cmd.Output()
	
	res := cleanID
	if err == nil {
		res = strings.TrimSpace(string(out))
	}

	// Salva no cache
	mu.Lock()
	if _, ok := cacheData[langCode]; !ok { cacheData[langCode] = make(map[string]string) }
	cacheData[langCode][normID] = res
	mu.Unlock()

	if verboseFlag {
		logger.Printf("%s (%s)\t%s [len=%-3d] %s %s => %s\n", 
			orange("[TRANS]"), cyan(langCode), yellow("Traduzindo:"), len(cleanID), white(cleanID), green("to"), blue(res))
	}

	return `"` + strings.ReplaceAll(res, `"`, `\"`) + `"`
}

func prepareGettext(input string) {
	potFile := input + ".pot"
	logger.Printf("%s Preparando: %s\n", black("[XGETTEXT]"), magenta(potFile))
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, input).Run()
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", potFile).Run()
}

func prepareMsginit(input, lang string) {
	potFile := input + ".pot"
	poTmp := fmt.Sprintf("%s-temp-%s.po", input, lang)
	logger.Printf("%s Inicializando idioma: '%s'\n", black("[MSGINIT]"), cyan(lang))
	exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	exec.Command("sed", "-i", "s/charset=ASCII/charset=utf-8/g", poTmp).Run()
}

func writeMsgfmtToMo(input, lang string) {
	dir := "usr/share/locale/" + lang + "/LC_MESSAGES"
	os.MkdirAll(dir, 0755)
	poFinal := fmt.Sprintf("%s-%s.po", input, lang)
	moFile := fmt.Sprintf("%s/%s.mo", dir, input)
	exec.Command("msgfmt", poFinal, "-o", moFile).Run()
	logger.Printf("%s Gerado: %s\n", green("[MSGFMT]"), magenta(moFile))
}

func usage() {
	fmt.Printf("\n%s\n%s\n\n", yellow("Chili Tradutor Go"), _COPY_)
	pflag.Usage()
}
