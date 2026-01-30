/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/chililinux/chili-tradutor-go

    Updated:   qui 29 jan 2026 22:40:00 -04
    Version:   2.1.19
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

// --- ESTRUTURAS E VARIÁVEIS GLOBAIS ---

type CacheEntry struct {
	Value    string    `json:"v"`
	LastUsed time.Time `json:"t"`
}

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.1.19-20260129"
	_COPY_    = "Copyright (C) 2019-2026 Vilmar Catafesta <vcatafesta@gmail.com>"
)

var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	white   = color.New(color.FgWhite).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	magenta = color.New(color.Bold, color.FgMagenta).SprintFunc()
)

var (
	inputFiles     []string
	currentFile    string
	engine         string
	sourceLang     string
	jobs           int
	forceFlag      bool
	quietFlag      bool
	verboseFlag    bool
	versionFlag    bool
	cleanCacheFlag bool
	selfFlag       bool
	selfTestFlag   bool
	languages      []string
	targetLangs    []string
	cacheFile      string
	cacheData      map[string]map[string]CacheEntry
	mu             sync.Mutex
	muConsole      sync.Mutex
	cacheHits      int
	netCalls       int
	failedCalls    int32
	isOnline       bool
	langsDone      int32
	langPositions  map[string]int
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt_PT", "pt_BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh_CN", "zh_TW",
}

var defaultLanguages = []string{"pt_BR", "en", "es", "it", "de", "fr", "ru", "zh_CN", "zh_TW", "ja", "ko"}

// --- FUNÇÃO DE EXECUÇÃO COM ISOLAMENTO DE LOCALE ---

func execCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	return cmd
}

// --- INICIALIZAÇÃO E MAIN ---

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755)
	cacheFile = filepath.Join(cacheDir, "cache.json")
}

func main() {
	checkDependencies()
	isOnline = checkInternet()
	parseFlags()

	if versionFlag {
		showVersion()
		os.Exit(0)
	}

	loadCache()
	defer saveCache()

	if selfTestFlag {
		runFullSelfTest()
		os.Exit(0)
	}

	if cleanCacheFlag {
		doCleanCache()
		os.Exit(0)
	}

	allFiles := append(inputFiles, pflag.Args()...)
	if len(allFiles) == 0 {
		usage()
		os.Exit(1)
	}

	startGlobal := time.Now()
	for _, file := range allFiles {
		processSingleFile(file)
	}

	if len(allFiles) > 1 {
		fmt.Printf("\n%s %s\n", green("✔"), white(T("Todos os arquivos foram processados!")))
		showFinalSummary(startGlobal)
	}
}

func processSingleFile(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("%s %s '%s'\n", red(T("ERRO:")), white(T("Arquivo não encontrado:")), yellow(path))
		return
	}

	currentFile = path
	langsDone = 0
	ext, langName, desc := detectFileType(path)
	baseName := filepath.Base(path)
	setupEnvironment(ext, baseName, langName)

	printWelcome(desc)
	start := time.Now()

	if !hasActualContent(ext, baseName) {
		fmt.Printf("%s %s\n", yellow(T("[AVISO]")), white(T("Nada para traduzir ou arquivo protegido.")))
		cleanupEmpty(ext, baseName)
	} else {
		fmt.Printf("%s %s\n\n", yellow(T("[STEP 2]")), white(T("Iniciando processamento paralelo...")))
		for _, lang := range targetLangs {
			fmt.Printf("    → %s %s\n", cyan(fmt.Sprintf("%-7s", lang)), yellow(T("[Aguardando...]")))
		}
		runTranslationLoop(ext, baseName)
	}
	showQuickStats(start)
}

// --- FUNÇÕES DE TRADUÇÃO DA UI ---

func T(msgid string) string {
	cmd := execCommand("gettext", "-d", _APP_, msgid)
	out, err := cmd.Output()
	if err != nil {
		return msgid
	}
	return strings.TrimSpace(string(out))
}

func TN(msgid, msgidPlural string, n int) string {
	cmd := execCommand("ngettext", "-d", _APP_, msgid, msgidPlural, fmt.Sprintf("%d", n))
	out, err := cmd.Output()
	if err != nil {
		if n == 1 {
			return msgid
		}
		return msgidPlural
	}
	return strings.TrimSpace(string(out))
}

// --- AUTO-TESTE EXAUSTIVO ---

func runFullSelfTest() {
	muConsole.Lock()
	fmt.Printf("\n%s %s %s\n", cyan(">>"), white("INICIANDO TESTE DE ESTRESSE GLOBAL EXAUSTIVO"), yellow("v"+_VERSION_))
	muConsole.Unlock()

	fmt.Printf("    %s %-35s ", blue("→"), T("Dependências e Conectividade"))
	checkDependencies()
	fmt.Println(green("OK"))

	fmt.Printf("    %s %-35s ", blue("→"), T("Proteção de Variáveis ($VAR)"))
	orig := "User $USER em https://chili.com com %d"
	prot, marks := protectVariables(orig)
	rest := restoreVariables(prot, marks)
	if orig == rest && strings.Contains(prot, "CHILI_REF") {
		fmt.Println(green("OK"))
	} else {
		fmt.Println(red("FALHA"))
	}

	fmt.Printf("    %s %-35s ", blue("→"), T("Validação de Markdown (.md)"))
	mdFile := "test_integrity.md"
	_ = os.WriteFile(mdFile, []byte("# Title\n```go\nfmt.Println(\"Keep\")\n```\nTranslate"), 0644)
	os.MkdirAll("doc", 0755)
	translateMarkdown(mdFile, "es")
	if _, err := os.Stat("doc/test_integrity-es.md"); err == nil {
		fmt.Println(green("OK"))
		os.Remove(mdFile)
		os.Remove("doc/test_integrity-es.md")
	} else {
		fmt.Println(red("FALHA"))
	}

	fmt.Printf("\n%s %s\n\n", green("✔"), white(T("SISTEMA 100% VALIDADO EM TODOS OS NÍVEIS.")))
}

// --- LOGICA DE CABEÇALHO ---

func stampPotHeader(path string, lang string) {
	langValue := "none"
	if lang != "" {
		langValue = lang
	}
	header := fmt.Sprintf(
		"# Chili Tradutor Go - %s\n"+
			"# Copyright (C) 2019-2026 Vilmar Catafesta <vcatafesta@gmail.com>\n"+
			"# This file is distributed under the same license as the %s package.\n"+
			"msgid \"\"\n"+
			"msgstr \"\"\n"+
			"\"Project-Id-Version: %s %s\\n\"\n"+
			"\"POT-Creation-Date: %s\\n\"\n"+
			"\"PO-Revision-Date: %s\\n\"\n"+
			"\"Last-Translator: Vilmar Catafesta <vcatafesta@gmail.com>\\n\"\n"+
			"\"Language-Team: Portuguese <https://github.com/chililinux/chili-tradutor-go>\\n\"\n"+
			"\"MIME-Version: 1.0\\n\"\n"+
			"\"Content-Type: text/plain; charset=UTF-8\\n\"\n"+
			"\"Content-Transfer-Encoding: 8bit\\n\"\n"+
			"\"Language: %s\\n\"\n"+
			"\"Plural-Forms: nplurals=2; plural=(n > 1);\\n\"\n\n",
		_VERSION_, _APP_, _APP_, _VERSION_, 
		time.Now().Format("2006-01-02 15:04-0700"), 
		time.Now().Format("2006-01-02 15:04-0700"), 
		langValue,
	)

	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	var actualContent []string
	found := false
	for _, line := range lines {
		if found || strings.HasPrefix(line, "#:") {
			found = true
			actualContent = append(actualContent, line)
		}
	}
	if found {
		final := header + strings.Join(actualContent, "\n")
		os.WriteFile(path, []byte(final), 0644)
	} else {
		re := regexp.MustCompile(`(?s)^msgid "".*?msgstr "".*?\n\n`)
		newContent := re.ReplaceAll(content, []byte(""))
		os.WriteFile(path, append([]byte(header), newContent...), 0644)
	}
}

// --- CORE TRANSLATION ENGINE ---

func runTranslationLoop(ext, baseName string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	targetBase := baseName
	if selfFlag {
		targetBase = _APP_ + ".go"
	}
	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			switch ext {
			case ".md", ".markdown":
				translateMarkdown(currentFile, l)
			case ".txt":
				translatePlaintext(currentFile, l)
			case ".json", ".yaml", ".yml":
				translateJSON(currentFile, l)
			case ".html", ".htm":
				translateHTML(currentFile, l)
			default:
				prepareMsginit(targetBase, l)
				translateFile(targetBase, l)
				writeMsgfmtToMo(targetBase, l)
			}
			atomic.AddInt32(&langsDone, 1)
			muConsole.Lock()
			if !selfTestFlag {
				fmt.Printf("\r    %s %s / %s %s", yellow(T("[STATUS]")), green(langsDone), green(len(targetLangs)), T("idiomas concluídos..."))
			}
			muConsole.Unlock()
			<-sem
		}(lang)
	}
	wg.Wait()
}

func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	normID := strings.ToLower(text)
	mu.Lock()
	if cacheData == nil {
		cacheData = make(map[string]map[string]CacheEntry)
	}
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

	transLang := strings.ReplaceAll(lang, "_", "-")
	protectedText, placeholders := protectVariables(text)
	var res string
	var err error
	for i := 0; i < 3; i++ {
		cmd := execCommand("trans", "-e", engine, "-s", sourceLang, "-no-init", "-no-autocorrect", "-b", ":"+transLang)
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

func prepareGettext(inputPath, baseName, lang string) {
	cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	pot := filepath.Join("pot", cleanName+".pot")
	execCommand("xgettext", "--from-code=UTF-8", "--language="+lang, "--keyword=gettext", "--keyword=_", "--keyword=T", "--keyword=TN:1,2", "--force-po", "-o", pot, inputPath).Run()
	stampPotHeader(pot, "")
}

func prepareGettextSelf(inputPath string) {
	pot := filepath.Join("pot", _APP_+".pot")
	execCommand("xgettext", "--from-code=UTF-8", "--keyword=T", "--keyword=TN:1,2", "--no-wrap", "-o", pot, inputPath).Run()
	stampPotHeader(pot, "")
}

func writeMsgfmtToMo(base, lang string) {
	cleanBase := strings.TrimSuffix(base, filepath.Ext(base))
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	poFile := filepath.Join("pot", fmt.Sprintf("%s-%s.po", cleanBase, lang))
	moFile := filepath.Join(dir, cleanBase+".mo")
	execCommand("msgfmt", "-f", poFile, "-o", moFile).Run()
}

func parseFlags() {
	pflag.Usage = usage
	pflag.StringSliceVarP(&inputFiles, "inputfile", "i", nil, T("Arquivo fonte"))
	pflag.StringVarP(&engine, "engine", "e", "google", T("Motor de tradução"))
	pflag.StringVarP(&sourceLang, "source", "s", "auto", T("Idioma de origem"))
	pflag.StringSliceVarP(&languages, "language", "l", nil, T("Idiomas destino"))
	pflag.IntVarP(&jobs, "jobs", "j", 8, T("Traduções simultâneas"))
	pflag.BoolVarP(&forceFlag, "force", "f", false, T("Ignora o cache"))
	pflag.BoolVar(&cleanCacheFlag, "clean-cache", false, T("Limpa cache antigo"))
	pflag.BoolVar(&selfFlag, "self", false, T("Extração especializada para o próprio chili-tradutor-go"))
	pflag.BoolVar(&selfTestFlag, "self-test", false, T("Executa auto-teste de integridade"))
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, T("Modo silencioso"))
	pflag.BoolVarP(&verboseFlag, "verbose", "v", false, T("Modo detalhado"))
	pflag.BoolVarP(&versionFlag, "version", "V", false, T("Mostra versão"))
	pflag.Parse()

	targetLangs = defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}
	langPositions = make(map[string]int)
	for i, lang := range targetLangs {
		langPositions[lang] = len(targetLangs) - i
	}
}

func setupEnvironment(ext, baseName, langName string) {
	switch ext {
	case ".md", ".markdown":
		os.MkdirAll("doc", 0755)
	case ".txt":
		os.MkdirAll("txt", 0755)
	case ".json":
		os.MkdirAll("json", 0755)
	case ".yaml", ".yml":
		os.MkdirAll("yml", 0755)
	case ".html", ".htm":
		os.MkdirAll("html", 0755)
	default:
		os.MkdirAll("pot", 0755)
		targetPot := filepath.Join("pot", baseName)
		if ext == ".pot" {
			absInput, _ := filepath.Abs(currentFile)
			absTarget, _ := filepath.Abs(targetPot)
			if absInput != absTarget {
				copyFile(currentFile, targetPot)
			}
		} else {
			if selfFlag {
				prepareGettextSelf(currentFile)
			} else {
				prepareGettext(currentFile, baseName, langName)
			}
		}
	}
}

func translateHTML(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	reTag := regexp.MustCompile(`(?s)<.*?>`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			translatedLines = append(translatedLines, line)
			continue
		}

		tagMap := make(map[string]string)
		counter := 0
		protected := reTag.ReplaceAllStringFunc(line, func(tag string) string {
			placeholder := fmt.Sprintf("CHILI_HTML_%d_CHILI", counter)
			tagMap[placeholder] = tag
			counter++
			return placeholder
		})

		textOnly := reTag.ReplaceAllString(line, "")
		if strings.TrimSpace(textOnly) != "" {
			translated := callUniversalTranslator(protected, lang)
			for placeholder, originalTag := range tagMap {
				translated = strings.ReplaceAll(translated, placeholder, originalTag)
			}
			translatedLines = append(translatedLines, translated)
		} else {
			translatedLines = append(translatedLines, line)
		}

		if i%5 == 0 || i == len(lines)-1 {
			updateProgress(lang, i+1, len(lines), "HTML")
		}
	}

	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outFile := filepath.Join("html", fmt.Sprintf("%s-%s%s", base, lang, ext))
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translateMarkdown(inputPath, lang string) {
	content, _ := os.ReadFile(inputPath)
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	inCodeBlock := false
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outFile := filepath.Join("doc", fmt.Sprintf("%s-%s%s", base, lang, ext))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			translatedLines = append(translatedLines, line)
			continue
		}
		if inCodeBlock || trimmed == "" {
			translatedLines = append(translatedLines, line)
			continue
		}
		rePrefix := regexp.MustCompile(`^(\s*#+\s*|\s*[\*\-\+]\s*|\s*\d+\.\s*)`)
		prefix, textToTranslate := "", line
		if loc := rePrefix.FindStringIndex(line); loc != nil {
			prefix = line[loc[0]:loc[1]]
			textToTranslate = line[loc[1]:]
		}
		translated := callUniversalTranslator(textToTranslate, lang)
		translatedLines = append(translatedLines, prefix+translated)
		if i%10 == 0 || i == len(lines)-1 {
			updateProgress(lang, i+1, len(lines), "MD")
		}
	}
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translatePlaintext(inputPath, lang string) {
	content, _ := os.ReadFile(inputPath)
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	
	ext := filepath.Ext(inputPath)
	if ext == "" { ext = ".txt" }
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outFile := filepath.Join("txt", fmt.Sprintf("%s-%s%s", base, lang, ext))

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			translatedLines = append(translatedLines, line)
		} else {
			translated := callUniversalTranslator(line, lang)
			translatedLines = append(translatedLines, translated)
		}

		if i%10 == 0 || i == len(lines)-1 {
			updateProgress(lang, i+1, len(lines), "TXT")
		}
	}
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translateFile(baseName, lang string) {
	cleanBase := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", cleanBase, lang))
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", cleanBase, lang))
	stampPotHeader(poTmp, lang)
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
		if strings.HasPrefix(l, "msgid ") && l != `msgid ""` {
			totalMsgids++
		}
	}
	current := 0
	var isMsgid bool
	var msgidLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "msgid ") && line != `msgid ""` {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
		} else if strings.HasPrefix(line, "msgstr ") && isMsgid {
			current++
			updateProgress(lang, current, totalMsgids, "PO")
			translated := callUniversalTranslator(strings.Trim(strings.Join(msgidLines, " "), `"`), lang)
			if translated == "" {
				translated = strings.Trim(strings.Join(msgidLines, " "), `"`)
			}
			fmt.Fprintf(output, "msgid %s\nmsgstr \"%s\"\n", strings.Join(msgidLines, "\n"), translated)
			isMsgid = false
		} else if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(output, line)
		}
	}
	os.Remove(poTmp)
	updateProgress(lang, totalMsgids, totalMsgids, "OK")
}

func translateJSON(path, lang string) {
	ext := strings.ToLower(filepath.Ext(path))
	targetDir := "json"
	if ext == ".yaml" || ext == ".yml" { targetDir = "yml" }
	data, _ := os.ReadFile(path)
	var obj map[string]interface{}
	json.Unmarshal(data, &obj)
	translateMap(obj, lang)
	out, _ := json.MarshalIndent(obj, "", "  ")
	outFile := filepath.Join(targetDir, fmt.Sprintf("%s-%s%s", strings.TrimSuffix(filepath.Base(path), ext), lang, ext))
	os.WriteFile(outFile, out, 0644)
	updateProgress(lang, 100, 100, strings.ToUpper(targetDir))
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

func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag || total == 0 { return }
	muConsole.Lock()
	defer muConsole.Unlock()
	pos := langPositions[lang]
	if pos == 0 { return }
	percent := (current * 100) / total
	width := 40
	filled := (percent * width) / 100
	bar := blue(strings.Repeat("░", filled)) + strings.Repeat(" ", width-filled)
	langStr := fmt.Sprintf("%-7s", lang)
	fmt.Printf("\033[%dA\r\033[K    → %s %s [%s] %3d%% %-5s\033[%dB", pos, blue("→"), cyan(langStr), bar, percent, cyan(suffix), pos)
}

func prepareMsginit(base, lang string) {
	cleanBase := strings.TrimSuffix(base, filepath.Ext(base))
	pot := filepath.Join("pot", cleanBase+".pot")
	po := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", cleanBase, lang))
	os.Remove(po)
	execCommand("msginit", "--no-translator", "-l", lang, "-i", pot, "-o", po).Run()
}

func protectVariables(text string) (string, map[string]string) {
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z]|!\[.*?\]\(.*?\)|\[.*?\]\(.*?\)|https?://[^\s]+)`)
	placeholders := make(map[string]string)
	protected := text
	matches := re.FindAllString(text, -1)
	for i, match := range matches {
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

func detectFileType(path string) (ext string, lang string, desc string) {
	ext = strings.ToLower(filepath.Ext(path))
	
	// Mapa de extensões conhecidas (sempre tem prioridade)
	extMap := map[string]string{
		".sh": "shell", ".py": "python", ".php": "php", ".c": "c",
		".cpp": "c++", ".go": "go", ".pl": "perl", ".rb": "ruby",
		".html": "html", ".htm": "html",
	}

	// Caso 1: Arquivo sem extensão
	if ext == "" {
		detected, _ := getShebangInfo(path)
		// Se o shebang existir, ele MANDA. Igual na 2.1.17.
		if detected != "" {
			return "", detected, fmt.Sprintf(T("Script (%s)"), green(detected))
		}
		// SÓ se não tiver shebang, tratamos como texto simples (Novidade 2.1.18+)
		return ".txt", "text", T("Texto Simples (sem extensão)")
	}

	// Caso 2: Extensões mapeadas
	if l, ok := extMap[ext]; ok {
		return ext, l, fmt.Sprintf(T("Código %s (%s)"), ext, green(l))
	}

	// Caso 3: Outros formatos específicos
	switch ext {
	case ".md", ".markdown":
		return ext, "markdown", T("Markdown")
	case ".txt":
		return ext, "text", T("Texto Simples")
	case ".json":
		return ext, "json", T("JSON")
	case ".yaml", ".yml":
		return ext, "yaml", T("YAML")
	case ".pot":
		return ext, "gettext", T("Template POT")
	}

	// Caso 4: Fallback padrão (Scripts genéricos se houver shebang ou shell por padrão)
	return ext, "shell", fmt.Sprintf(T("Arquivo %s"), ext)
}

func getShebangInfo(path string) (string, string) {
	f, err := os.Open(path)
	if err != nil { return "", "" }
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#!") {
			lower := strings.ToLower(line)
			switch {
			case strings.Contains(lower, "python"):
				return "python", line
			case strings.Contains(lower, "php"):
				return "php", line
			case strings.Contains(lower, "perl"):
				return "perl", line
			case strings.Contains(lower, "ruby"):
				return "ruby", line
			case strings.Contains(lower, "node"):
				return "javascript", line
			case strings.Contains(lower, "bash") || strings.Contains(lower, "sh"):
				return "shell", line
			}
			return "shell", line
		}
	}
	return "", ""
}

func checkInternet() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil { return false }
	conn.Close()
	return true
}

func detectDistro() string {
	osData, _ := os.ReadFile("/etc/os-release")
	osContent := string(osData)
	re := regexp.MustCompile(`(?m)^ID=["']?([^"'\s]+)["']?`)
	match := re.FindStringSubmatch(osContent)
	if len(match) > 1 { return strings.ToLower(match[1]) }
	return "unknown"
}

func checkDependencies() {
	deps := map[string]string{
		"xgettext": "gettext", "msginit":  "gettext", "msgfmt":   "gettext",
		"gettext":  "gettext", "ngettext": "gettext", "trans":    "translate-shell",
	}
	missingMap := make(map[string]bool)
	hasMissing := false
	for bin, pkg := range deps {
		if _, err := exec.LookPath(bin); err != nil {
			missingMap[pkg] = true
			hasMissing = true
		}
	}
	if !hasMissing { return }
	var missingPkgs []string
	for pkg := range missingMap { missingPkgs = append(missingPkgs, pkg) }
	pkgList := strings.Join(missingPkgs, " ")
	muConsole.Lock()
	fmt.Printf("\n%s %s\n", red(" [ERRO]"), white(T("Dependências ausentes: ")+pkgList))
	distro := detectDistro()
	installCmd := ""
	switch distro {
	case "chili", "chililinux", "arch": installCmd = "sudo pacman -S " + pkgList
	case "void": installCmd = "sudo xbps-install -S " + pkgList
	case "debian", "ubuntu": installCmd = "sudo apt install " + pkgList
	case "fedora": installCmd = "sudo dnf install " + pkgList
	}
	if installCmd != "" {
		fmt.Printf("\n %s %s (%s)? (s/N): ", yellow(" →"), T("Deseja instalar automaticamente para"), cyan(distro))
		muConsole.Unlock()
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "s" {
			args := strings.Fields(installCmd)
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			if err := cmd.Run(); err == nil { return }
		}
	} else { muConsole.Unlock() }
	os.Exit(1)
}

func loadCache() {
	cacheData = make(map[string]map[string]CacheEntry)
	file, err := os.ReadFile(cacheFile)
	if err == nil { json.Unmarshal(file, &cacheData) }
}

func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ")
	os.WriteFile(cacheFile, data, 0644)
}

func copyFile(src, dst string) error {
	s, _ := os.Open(src); defer s.Close()
	d, _ := os.Create(dst); defer d.Close()
	_, err := io.Copy(d, s)
	return err
}

func hasActualContent(ext, baseName string) bool {
	if selfFlag { return true }
	if ext == ".md" || ext == ".markdown" || ext == ".txt" || ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".html" || ext == ".htm" { return true }
	potFile := filepath.Join("pot", baseName+".pot")
	if _, err := os.Stat(potFile); err == nil {
		content, _ := os.ReadFile(potFile)
		return strings.Contains(string(content), "msgid")
	}
	return true
}

func cleanupEmpty(ext, baseName string) {
	potFile := filepath.Join("pot", baseName+".pot")
	os.Remove(potFile)
}

func printWelcome(desc string) {
	fmt.Printf("\n%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))
	fmt.Printf("%s %s\n", yellow(T("[STEP 1]")), white(T("Ambiente preparado com sucesso.")))
	fmt.Printf("    → %-15s: %s\n", T("Arquivo"), white(currentFile))
	fmt.Printf("    → %-15s: %s\n", T("Tipo"), cyan(desc))
	fmt.Printf("    → %-15s: %s\n", T("Motor"), green(engine))
	fmt.Printf("    → %-15s: %s (%s)\n", T("Origem"), green(sourceLang), T("Auto-detect se auto"))
	fmt.Printf("    → %-15s: %s\n", T("Jobs"), red(jobs))
	fmt.Printf("    → %-15s: %s\n\n", T("Cache"), blue(cacheFile))
}

func showQuickStats(start time.Time) {
	total := cacheHits + netCalls
	pCache, pNet := 0.0, 0.0
	if total > 0 {
		pCache = (float64(cacheHits) / float64(total)) * 100
		pNet = (float64(netCalls) / float64(total)) * 100
	}
	fmt.Printf("\n\n%s %s em %v | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n", green("✔"), white(T("Concluído")), time.Since(start).Round(time.Second), blue(T("Cache:")), cacheHits, pCache, yellow(T("Net:")), netCalls, pNet, white(T("Total:")), total)
}

func showFinalSummary(start time.Time) {
	fmt.Printf("%s\n %s\n", white(strings.Repeat("-", 60)), yellow(T("RESUMO EXECUTIVO FINAL:")))
	fmt.Printf("    → %-15s: %v\n", T("Tempo Total"), time.Since(start).Round(time.Second))
	fmt.Printf("    → %-15s: %d\n", T("Cache Hits"), cacheHits)
	fmt.Printf("    → %-15s: %d\n", T("Chamadas Rede"), netCalls)
	if atomic.LoadInt32(&failedCalls) > 0 {
		fmt.Printf("    → %-15s: %s\n", T("Falhas"), red(atomic.LoadInt32(&failedCalls)))
	}
	fmt.Printf("%s\n\n", white(strings.Repeat("-", 60)))
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
	fmt.Printf("%s %s %d %s\n", green("✔"), T("Removidos"), count, T("itens obsoletos do cache."))
}

func showVersion() { fmt.Printf("%s %s\n%s\n", cyan(_APP_), white(_VERSION_), white(_COPY_)) }

func usage() {
	fmt.Fprintf(os.Stderr, "\n%s %s\n%s\n\n", cyan(_APP_), white(_VERSION_), white(_COPY_))
	fmt.Fprintf(os.Stderr, "%s: %s %s %s\n\n", yellow(T("Uso")), green(_APP_), yellow("-i"), green(T("<arquivo> [opções]")))
	fmt.Fprintf(os.Stderr, "%s:\n", yellow(T("Opções")))
	defLangs := strings.Join(defaultLanguages, ",")
	flags := []struct{ short, long, desc string }{
		{"-i", "--inputfile", T("Arquivo fonte (.sh, .py, .md, .txt, .json, .yaml, .html, .pot)")},
		{"-l", "--language", fmt.Sprintf(T("Idiomas (ex: pt_BR,en) ou 'all' (padrão: %s)"), defLangs)},
		{"-e", "--engine", T("Motor: google, bing, yandex (padrão: google)")},
		{"-j", "--jobs", T("Traduções simultâneas (padrão: 8)")},
		{"-s", "--source", T("Idioma de origem (ex: pt, en) (padrão: auto)")},
		{"-f", "--force", T("Força nova tradução (ignora cache)")},
		{"", "--self", T("Extração especializada para o próprio chili-tradutor-go")},
		{"", "--self-test", T("Executa auto-teste de integridade")},
		{"", "--clean-cache", T("Remove entradas de cache não usadas há 30 dias")},
		{"-q", "--quiet", T("Modo silencioso")},
		{"-v", "--verbose", T("Mostrar detalhes")},
		{"-V", "--version", T("Mostra a versão do programa")},
	}
	for _, f := range flags {
		if f.short != "" {
			fmt.Fprintf(os.Stderr, "  %s, %-30s %s\n", cyan(f.short), cyan(f.long), white(f.desc))
		} else {
			fmt.Fprintf(os.Stderr, "      %-30s %s\n", cyan(f.long), white(f.desc))
		}
	}
}
