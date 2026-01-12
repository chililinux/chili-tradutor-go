/*
   chili-tradutor-go - v1.7.0
   Copyright (c) 2023-2026, Vilmar Catafesta <vcatafesta@gmail.com>
   Licença: BSD-2-Clause
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
	"time"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "1.7.0-20260110"
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
	// Estatísticas
	cacheHits    int
	netCalls     int
	isOnline     bool
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh",
}

var defaultLanguages = []string{"de", "it", "en", "es", "fr"}

func init() {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755)
	cacheFile = filepath.Join(cacheDir, "cache.json")
}

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

func checkInternet() bool {
	timeout := 2 * time.Second
	_, err := net.DialTimeout("tcp", "google.com:80", timeout)
	return err == nil
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

// Proteção de variáveis Shell (ex: $USER, ${VAR}, %s)
func protectVariables(text string) (string, map[string]string) {
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z])`)
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

func main() {
	checkDependencies()
	isOnline = checkInternet() // Verifica se há conexão com a rede no início

	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor de tradução")
	pflag.IntVarP(&jobs, "jobs", "j", 3, "Traduções simultâneas")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar tradução (ignorar cache)")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo quieto")
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, "Mostrar detalhes de cada ID")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas (ou 'all' para todos)")
	
	pflag.Usage = usage // Define nosso usage colorido customizado
	pflag.Parse()

	if inputFile == "" {
		usage()
		os.Exit(1)
	}

	confLog()
	loadCache()
	defer saveCache()

	// Lógica de seleção de idiomas: Top 5 vs All vs Lista Manual
	targetLangs := defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}

	// --- CABEÇALHO VISUAL ---
	fmt.Printf("%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))
	
	connStatus := green("ONLINE")
	if !isOnline {
		connStatus = red("OFFLINE (Usando apenas Cache)")
	}

	fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Extraindo strings..."))
	prepareGettext(inputFile)

	fmt.Printf("%s %s\n", yellow("[STEP 2]"), white("Iniciando processamento de tradução"))
	fmt.Printf("%s %-25s %d\n", magenta("[LANGS]"), white("Idiomas selecionados:"), len(targetLangs))
	fmt.Printf("%s %-25s %s\n", magenta("[CONN]"), white("Status da rede:"), connStatus)
	fmt.Printf("%s %-25s %d\n", magenta("[JOBS]"), white("Processos paralelos:"), jobs)
	fmt.Printf("%s %-25s %s\n\n", magenta("[ENGINE]"), white("Motor de tradução:"), cyan(engine))

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)

	// Processamento paralelo por idioma
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

   // --- RELATÓRIO FINAL COM ESTATÍSTICAS COMPLETAS ---
	duration := time.Since(start)
	totalRequests := cacheHits + netCalls
	cachePercent := 0.0
	netPercent := 0.0

	if totalRequests > 0 {
		cachePercent = (float64(cacheHits) / float64(totalRequests)) * 100
		netPercent = (float64(netCalls) / float64(totalRequests)) * 100
	}

	fmt.Printf("\n%s %s\n", green("✔"), white("Processo concluído com sucesso!"))
	fmt.Printf("%s %s: %d | %s: %d (%s) | %s: %d (%s) | %s: %v\n", 
		magenta("[STATS]"), 
		white("Total"), totalRequests,
		white("Cache Hits"), cacheHits, cyan(fmt.Sprintf("%.1f%%", cachePercent)),
		white("Net Calls"), netCalls, orange(fmt.Sprintf("%.1f%%", netPercent)),
		white("Tempo"), duration.Round(time.Second))
}

func translateFile(inputFile, lang string) {
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	poFinal := fmt.Sprintf("%s-%s.po", inputFile, lang)
	file, err := os.Open(poTmp)
	if err != nil { return }
	defer file.Close()
	output, _ := os.Create(poFinal)
	defer output.Close()

	logger.Printf("%s Traduzindo idioma: '%s'\n", black("[TRANS]"), cyan(lang))

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
}

func translateMessage(msgid, langCode string) string {
	msgid = strings.TrimSpace(msgid)
	cleanID := strings.Trim(msgid, `"`)
	if cleanID == "" { return "msgstr \"\"" }

	normID := strings.ToLower(strings.TrimSpace(cleanID))
	
	mu.Lock()
	if val, ok := cacheData[langCode][normID]; ok && !forceFlag {
		cacheHits++
		mu.Unlock()
		if verboseFlag { logger.Printf("   -> (%s) ID: %s [CACHE]\n", cyan(langCode), white(cleanID)) }
		return formatMsgstr(val)
	}
	mu.Unlock()

	if !isOnline { return formatMsgstr(cleanID) } // Se offline e sem cache, mantém original

	// Protege variáveis antes de enviar para a web
	protectedText, placeholders := protectVariables(cleanID)
	
	netCalls++
	if verboseFlag { logger.Printf("   -> (%s) ID: %s [NET]\n", cyan(langCode), white(cleanID)) }
	
	cmd := exec.Command("trans", "-e", engine, "-no-autocorrect", "-b", ":"+langCode, protectedText)
	out, err := cmd.Output()
	
	res := protectedText
	if err == nil {
		res = strings.TrimSpace(string(out))
	}

	// Restaura variáveis após tradução
	res = restoreVariables(res, placeholders)

	mu.Lock()
	if _, ok := cacheData[langCode]; !ok { cacheData[langCode] = make(map[string]string) }
	cacheData[langCode][normID] = res
	mu.Unlock()

	return formatMsgstr(res)
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

func prepareGettext(input string) {
	potFile := input + ".pot"
	logger.Printf("%s Preparando: %s\n", black("[XGETTEXT]"), magenta(potFile))
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, input).Run()
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", potFile).Run()
}

func prepareMsginit(input, lang string) {
	potFile := input + ".pot"
	poTmp := fmt.Sprintf("%s-temp-%s.po", input, lang)
	logger.Printf("%s Rodando msginit: '%s'\n", black("[MSGINIT]"), cyan(lang))
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

func confLog() {
	fileLog := "/tmp/" + _APP_ + ".log"
	logFile, _ := os.OpenFile(fileLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if quietFlag { logger = log.New(logFile, "", 0) } else { logger = log.New(io.MultiWriter(os.Stdout, logFile), "", 0) }
}

func usage() {
	fmt.Printf("\n%s %s\n", yellow(">>"), white(_APP_))
	fmt.Printf("%s %s\n", yellow(">>"), white(_COPY_))
	fmt.Printf("%s %s\n\n", yellow(">>"), white(_VERSION_))
	fmt.Printf("%s\n", boldred("USO:"))
	fmt.Printf("  %s %s %s %s\n\n", cyan(_APP_), yellow("-i"), green("<arquivo>"), magenta("[opções]"))
	fmt.Printf("%s\n", boldred("OPÇÕES:"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-i"), yellow("--inputfile"), white("Arquivo de entrada"))
	fmt.Printf("  %s, %-15s %s %s\n", yellow("-l"), yellow("--language"), white("Idiomas ou"), magenta("'all'"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-f"), yellow("--force"), white("Ignorar cache"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-v"), yellow("--verbose"), white("Detalhar tradução"))
	fmt.Printf("  %s, %-15s %s\n", yellow("-h"), yellow("--help"), white("Mostrar ajuda"))
}
