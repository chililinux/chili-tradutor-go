/*
   chili-tradutor-go - v1.6.0
   Copyright (c) 2023-2026, Vilmar Catafesta <vcatafesta@gmail.com>
   Merge: Funcionalidades 1.5.8 + Estética e Dependências da versão enviada.
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
	_VERSION_ = "1.6.0-20260110"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	orange  = color.New(color.FgYellow).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	magenta = color.New(color.Bold, color.FgMagenta).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	black   = color.New(color.Bold, color.FgBlack).SprintFunc()
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

// Resgatado da sua versão enviada: Garante que o sistema tem o necessário
func checkDependencies() {
	required := []string{"xgettext", "msginit", "msgfmt", "sed", "trans"}
	missing := []string{}
	for _, bin := range required {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		log.Fatalf("Dependências faltando: %s\n", strings.Join(missing, ", "))
	}
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
	checkDependencies()

	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor de tradução")
	pflag.IntVarP(&jobs, "jobs", "j", 3, "Traduções simultâneas")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar tradução")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo quieto")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, "Verbose")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas")
	pflag.Parse()

	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	confLog()
	loadCache()
	defer saveCache()

	targetLangs := supportedLanguages
	if len(languages) > 0 {
		targetLangs = languages
	}

	prepareGettext(inputFile)

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
	logger.Printf("\n%s %s\n", green("✔"), white("Processo concluído!"))
}

func translateFile(inputFile, lang string) {
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	poFinal := fmt.Sprintf("%s-%s.po", inputFile, lang)

	file, err := os.Open(poTmp)
	if err != nil { return }
	defer file.Close()

	output, _ := os.Create(poFinal)
	defer output.Close()

	logger.Printf("%s Traduzindo mensagens para o idioma: '%s'\n", black("[TRANS]"), cyan(lang))

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
			msgstrFormatted := translateMessage(fullMsgid, lang)
			fmt.Fprintf(output, "msgid %s\n%s\n", strings.Join(msgidLines, "\n"), msgstrFormatted)
			isMsgid = false
		} else if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(output, line)
		}
	}
	logger.Printf("%s Traduzido para o idioma: '%s' [%s]\n", green("[TRANS]"), cyan(lang), poFinal)
}

func translateMessage(msgid, langCode string) string {
	msgid = strings.TrimSpace(msgid)
	cleanID := strings.Trim(msgid, `"`)
	if cleanID == "" { return "msgstr \"\"" }

	// Cache
	normID := strings.ToLower(strings.TrimSpace(cleanID))
	mu.Lock()
	if val, ok := cacheData[langCode][normID]; ok && !forceFlag {
		mu.Unlock()
		if verboseFlag {
			logger.Printf("   %s (%s) %s %s [CACHE]\n", orange("->"), cyan(langCode), yellow("ID:"), white(cleanID))
		}
		return formatMsgstr(val)
	}
	mu.Unlock()

	// Tradução
	if verboseFlag {
		logger.Printf("   %s (%s) %s %s [NET]\n", orange("->"), cyan(langCode), yellow("ID:"), white(cleanID))
	}
	
	cmd := exec.Command("trans", "-e", engine, "-no-autocorrect", "-b", ":"+langCode, cleanID)
	out, err := cmd.Output()
	
	res := cleanID
	if err == nil {
		res = strings.TrimSpace(string(out))
	}

	mu.Lock()
	if _, ok := cacheData[langCode]; !ok { cacheData[langCode] = make(map[string]string) }
	cacheData[langCode][normID] = res
	mu.Unlock()

	return formatMsgstr(res)
}

// Formatação robusta para PO (suporta múltiplas linhas)
func formatMsgstr(text string) string {
	text = strings.ReplaceAll(text, `"`, `\"`)
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		return fmt.Sprintf("msgstr \"%s\"", lines[0])
	}
	res := "msgstr \"\"\n"
	for i, l := range lines {
		if i < len(lines)-1 {
			res += fmt.Sprintf("\"%s\\n\"\n", l)
		} else {
			res += fmt.Sprintf("\"%s\"", l)
		}
	}
	return res
}

func prepareGettext(input string) {
	potFile := input + ".pot"
	logger.Printf("%s Preparando arquivo: %s\n", black("[XGETTEXT]"), magenta(potFile))
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, input).Run()
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", potFile).Run()
}

func prepareMsginit(input, lang string) {
	potFile := input + ".pot"
	poTmp := fmt.Sprintf("%s-temp-%s.po", input, lang)
	logger.Printf("%s Rodando msginit para o idioma: '%s'\n", black("[MSGINIT]"), cyan(lang))
	exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	exec.Command("sed", "-i", "s/charset=ASCII/charset=utf-8/g", poTmp).Run()
}

func writeMsgfmtToMo(input, lang string) {
	dir := "usr/share/locale/" + lang + "/LC_MESSAGES"
	os.MkdirAll(dir, 0755)
	poFinal := fmt.Sprintf("%s-%s.po", input, lang)
	moFile := fmt.Sprintf("%s/%s.mo", dir, input)
	exec.Command("msgfmt", poFinal, "-o", moFile).Run()
	logger.Printf("%s Traduzido para o idioma '%s':\t%s\n", green("[MSGFMT]"), cyan(lang), magenta(moFile))
}

func usage() {
	fmt.Printf("\n%s\n%s\n\n", yellow("Chili Tradutor Go"), _COPY_)
	pflag.Usage()
}
