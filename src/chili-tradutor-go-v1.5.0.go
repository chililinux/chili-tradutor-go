/*
   chili-tradutor-go - v1.5.1
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
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "1.5.1-20260110"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	green   = color.New(color.Bold, color.FgGreen).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	red     = color.New(color.Bold, color.FgRed).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	black   = color.New(color.Bold, color.FgBlack).SprintFunc()
)

var (
	inputFile    string
	engine       string
	jobs         int
	showVersion  bool
	showHelp     bool
	forceFlag    bool
	quietFlag    bool
	installFlag  bool
	languages    []string
	logger       *log.Logger
	cacheFile    string
	cacheData    map[string]map[string]string
	mu           sync.Mutex
)

func init() {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755)
	cacheFile = filepath.Join(cacheDir, "cache.json")
	cacheData = make(map[string]map[string]string)
}

func loadCache() {
	file, err := os.ReadFile(cacheFile)
	if err == nil {
		_ = json.Unmarshal(file, &cacheData)
	}
}

func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ")
	_ = os.WriteFile(cacheFile, data, 0644)
}

func main() {
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo .sh de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor (google, bing, yandex)")
	pflag.IntVarP(&jobs, "jobs", "j", 3, "Traduções simultâneas")
	pflag.BoolVarP(&installFlag, "install", "I", false, "Instalar no sistema")
	pflag.BoolVarP(&showVersion, "version", "V", false, "Versão")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar regeração")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo silencioso")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas")
	pflag.Parse()

	if showVersion {
		fmt.Println(_VERSION_)
		os.Exit(0)
	}
	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	loadCache()
	defer saveCache()

	// Configuração do Log e Saída
	fileLog, _ := os.OpenFile("/tmp/"+_APP_+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	var logOutput io.Writer
	if quietFlag {
		logOutput = fileLog
	} else {
		logOutput = io.MultiWriter(os.Stdout, fileLog)
	}
	logger = log.New(logOutput, "", 0)

	fmt.Printf("%s %s %s\n", cyan(">>"), green(_APP_), yellow(_VERSION_))
	
	if err := checkEngineHealth(engine); err != nil {
		logger.Printf("%s Motor %s com problemas, usará fallback se necessário.\n", yellow("[WARN]"), engine)
	}

	logger.Printf("%s Iniciando extração de strings (xgettext)...\n", black("[STEP 1]"))
	prepareGettext(inputFile)

	if len(languages) == 0 {
		languages = []string{"en", "es", "fr", "de", "it"}
	}

	logger.Printf("%s Traduzindo para %d idiomas com %d jobs...\n", black("[STEP 2]"), len(languages), jobs)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, jobs)

	for _, lang := range languages {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logger.Printf("%s Processando idioma: %s\n", cyan("[LANG]"), l)
			prepareMsginit(inputFile, l)
			translateFile(inputFile, l, engine)
			writeMsgfmtToMo(inputFile, l)
			
			if installFlag {
				installMo(l, inputFile)
			}
			os.Remove(fmt.Sprintf("%s-temp-%s.po", inputFile, l))
		}(lang)
	}
	wg.Wait()
	fmt.Printf("%s %s\n", green("✔"), "Processo concluído!")
}

// --- Funções Auxiliares (Mesma lógica da anterior, mas com logs visíveis) ---

func protectVars(text string) (string, map[string]string) {
	re := regexp.MustCompile(`(\$\{[A-Z0-9_]+\}|\$[A-Z0-9_]+)`)
	placeholders := make(map[string]string)
	matches := re.FindAllString(text, -1)
	for i, match := range matches {
		placeholder := fmt.Sprintf("___VAR%d___", i)
		placeholders[placeholder] = match
		text = strings.Replace(text, match, placeholder, 1)
	}
	return text, placeholders
}

func restoreVars(text string, placeholders map[string]string) string {
	for ph, original := range placeholders {
		text = strings.Replace(text, ph, original, -1)
	}
	return text
}

func getFromCache(lang, text string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if !forceFlag {
		if l, ok := cacheData[lang]; ok {
			if t, ok := l[text]; ok {
				return t, true
			}
		}
	}
	return "", false
}

func addToCache(lang, text, translation string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := cacheData[lang]; !ok {
		cacheData[lang] = make(map[string]string)
	}
	cacheData[lang][text] = translation
}

func translateMessage(msgid, lang, eng string) string {
	msgid = strings.Trim(strings.TrimSpace(msgid), `"`)
	if msgid == "" { return "msgstr \"\"" }

	if val, ok := getFromCache(lang, msgid); ok {
		return formatMsgstr(val)
	}

	protectedText, pMap := protectVars(msgid)
	engines := []string{eng, "bing", "yandex"}
	var translated string
	var success bool

	for _, e := range engines {
		cmd := exec.Command("trans", "-e", e, "-no-autocorrect", "-b", ":"+lang, protectedText)
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			translated = strings.TrimSpace(string(out))
			success = true
			break
		}
	}

	if !success { translated = protectedText }

	finalText := restoreVars(translated, pMap)
	finalText = strings.ReplaceAll(finalText, `"`, `\"`)
	addToCache(lang, msgid, finalText)

	return formatMsgstr(finalText)
}

func formatMsgstr(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 1 { return fmt.Sprintf("msgstr \"%s\"", lines[0]) }
	res := "msgstr \"\"\n"
	for i, l := range lines {
		suffix := "\\n"
		if i == len(lines)-1 { suffix = "" }
		res += fmt.Sprintf("\"%s%s\"\n", l, suffix)
	}
	return res
}

func checkEngineHealth(eng string) error {
	cmd := exec.Command("trans", "-e", eng, "-no-autocorrect", "-b", ":en", "health")
	return cmd.Run()
}

func prepareGettext(inputFile string) {
	potFile := inputFile + ".pot"
	_ = exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, inputFile).Run()
	_ = exec.Command("sed", "-i", `s/charset=CHARSET/charset=UTF-8/`, potFile).Run()
}

func prepareMsginit(inputFile, lang string) {
	potFile := inputFile + ".pot"
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	_ = exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	_ = exec.Command("sed", "-i", `s/charset=ASCII/charset=utf-8/g`, poTmp).Run()
}

func translateFile(inputFile, lang, eng string) {
	poFinal := fmt.Sprintf("%s-%s.po", inputFile, lang)
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	inFile, _ := os.Open(poTmp)
	defer inFile.Close()
	outFile, _ := os.Create(poFinal)
	defer outFile.Close()
	scanner := bufio.NewScanner(inFile)
	var isMsgid bool
	var msgidLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid ") {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
			continue
		}
		if strings.HasPrefix(line, "msgstr ") && isMsgid {
			fullMsgid := strings.Join(msgidLines, " ")
			fmt.Fprintf(outFile, "msgid %s\n%s\n", strings.Join(msgidLines, "\n"), translateMessage(fullMsgid, lang, eng))
			isMsgid = false
			continue
		}
		if isMsgid { msgidLines = append(msgidLines, line) } else { fmt.Fprintln(outFile, line) }
	}
}

func writeMsgfmtToMo(inputFile, lang string) {
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	poFile := fmt.Sprintf("%s-%s.po", inputFile, lang)
	moFile := filepath.Join(dir, inputFile+".mo")
	_ = exec.Command("msgfmt", poFile, "-o", moFile).Run()
}

func installMo(lang, inputFile string) {
	src := filepath.Join("usr/share/locale", lang, "LC_MESSAGES", inputFile+".mo")
	destDir := filepath.Join("/usr/share/locale", lang, "LC_MESSAGES")
	exec.Command("sudo", "mkdir", "-p", destDir).Run()
	_ = exec.Command("sudo", "cp", src, filepath.Join(destDir, inputFile+".mo")).Run()
}

func usage() {
	fmt.Printf("\n%s\n\n", yellow("Chili Tradutor Go"))
	pflag.Usage()
}
