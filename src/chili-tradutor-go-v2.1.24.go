/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/chililinux/chili-tradutor-go

    Updated:   dom 16 ago 2026 (patch de correções — ver CORRECOES.md)
    Version:   2.1.24

    Changelog 2.1.24 (nova varredura completa pós-2.1.23):
      - langsDone: leitura não-atômica misturada com atomic.AddInt32 concorrente
        (mesma classe do BUG-02, não pega na correção anterior) — agora usa
        atomic.LoadInt32 antes de imprimir
      - glossário: termo curto listado antes de um termo mais longo que o contém
        (ex: "Paulo" antes de "São Paulo") fazia a entrada mais específica falhar
        silenciosamente; regras agora ordenadas do termo mais longo para o mais curto
      - poPluralIndex: índice malformado em "msgstr[N]" podia ser rotulado como
        msgstr[0] e duplicar uma entrada existente; agora reproduz a linha original
        em vez de arriscar a duplicação

    Changelog 2.1.23 (itens 8-10 da varredura anterior):
      - item 8: msgid_plural/msgstr[N] (formas plurais do gettext) não eram traduzidos;
        parser de .po reescrito para reconhecer e traduzir ambas as formas
      - item 9: comentário do guard pos==0 em updateProgress corrigido (não é código
        morto — protege contra 'lang' desconhecido, já que map lookup em Go retorna
        zero-value; mantido, apenas documentado corretamente)
      - item 10: T()/TN() memoizadas — antes disparavam um subprocesso gettext/ngettext
        a CADA chamada (T() é chamada ~90 vezes no código), mesmo repetindo o msgid
        entre múltiplos arquivos processados na mesma execução

    Changelog 2.1.22 (correções sobre a 2.1.21, após varredura da rodada de features):
      - dryWriteFile() tinha erro ignorado em translateManPage/HTML/Markdown/Plaintext
      - --dry-run era parcial: setupEnvironment/prepareMsginit/translateFile ainda
        gravavam artefatos reais do pipeline gettext (.pot/.po) mesmo em modo simulação
      - variável 'net' em showQuickStats sombreava o pacote "net" importado
      - glossário: \b (RE2) falhava em termos que começam/terminam com letra acentuada
        (café, ação, não); substituído por checagem manual de fronteira Unicode
      - glossário: "termo=tradução" agora aceita também "termo=idioma:trad;idioma2:trad2"
        para traduções fixas diferentes por idioma-alvo
      - --dry-run não exercitava a proteção de glossário por substring (só o match exato)
      - acertos de glossário não entravam nas estatísticas de cache/rede exibidas

    Changelog 2.1.21 (correções sobre a 2.1.20):
      - BUG-01: --clean-cache e --self-test não persistiam o cache (os.Exit pulava defer)
      - BUG-02: data race em cacheHits/netCalls (agora atômicos)
      - BUG-03: -j <= 0 causava panic (make de channel com tamanho inválido)
      - BUG-04: .yaml/.yml eram parseados com encoding/json (agora usa yaml.v3)
      - BUG-05: strings dentro de arrays JSON/YAML não eram traduzidas
      - BUG-06: msgid multilinha em .po nunca era traduzido
      - BUG-07: estatísticas de cache/rede por arquivo mostravam total acumulado
      - BUG-08: erros de xgettext/msginit/msgfmt eram descartados silenciosamente
      - BUG-09: copyFile/translateFile ignoravam erros de Open/Create
      - BUG-10: regex de proteção de %s/%d não cobria %.2f, %5d, %-10s etc.
      - BUG-11: --verbose não tinha efeito nenhum (agora ativa logVerbose)
      - BUG-12: --quiet não suprimia printWelcome/showQuickStats/showFinalSummary
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
	"os/signal" // FEATURE: tratamento de SIGINT/SIGTERM (salva cache antes de encerrar)
	"path/filepath"
	"regexp"
	"sort" // BUGFIX: ordenação de glossaryRules por especificidade
	"strconv" // BUGFIX: parsing de msgstr[N] (formas plurais, item 8)
	"strings"
	"sync"
	"sync/atomic"
	"syscall" // FEATURE: idem
	"time"
	"unicode"     // BUGFIX: fronteira de palavra manual no glossário (termos acentuados)
	"unicode/utf8" // BUGFIX: idem

	"github.com/fatih/color"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3" // BUGFIX (BUG-04): parsing correto de YAML (antes usava encoding/json)
)

// --- ESTRUTURAS E VARIÁVEIS GLOBAIS ---

type CacheEntry struct {
	Value    string    `json:"v"`
	LastUsed time.Time `json:"t"`
}

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.1.24-20260816"
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
	dryRunFlag     bool // FEATURE: --dry-run
	glossaryPath   string
	glossary       map[string]glossaryEntry // FEATURE: --glossary (match exato do texto inteiro)
	glossaryRules  []glossaryRule           // FEATURE: --glossary (match de termo dentro de frases)
	languages      []string
	targetLangs    []string
	cacheFile      string
	cacheData      map[string]map[string]CacheEntry
	mu             sync.Mutex
	muConsole      sync.Mutex
	cacheHits      int64 // BUGFIX: era int com netCalls++ fora de lock (data race); agora atômico.
	netCalls       int64 // BUGFIX: idem.
	fileCacheHits  int64 // BUGFIX: contador por-arquivo, resetado a cada processSingleFile.
	fileNetCalls   int64 // BUGFIX: idem — antes showQuickStats exibia totais acumulados de todos os arquivos.
	glossaryHits   int64 // BUGFIX: contador de acertos de glossário, antes não entrava nas estatísticas.
	fileGlossaryHits int64 // BUGFIX: idem, por-arquivo.
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

// dryWriteFile grava o arquivo normalmente, exceto em --dry-run, onde apenas registra
// (via logVerbose) o que seria gravado, sem tocar o disco. Centraliza esse comportamento
// para todos os formatos de saída de documento (HTML, Markdown, TXT, JSON/YAML, man page).
func dryWriteFile(path string, data []byte) error {
	if dryRunFlag {
		logVerbose("[DRY-RUN] gravaria %d bytes em %s", len(data), path)
		return nil
	}
	return os.WriteFile(path, data, 0644)
}

func execCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	return cmd
}

// logVerbose imprime mensagens de depuração apenas quando -v/--verbose está ativa.
// BUGFIX (BUG-11): a flag --verbose existia na CLI mas não tinha efeito nenhum.
func logVerbose(format string, args ...interface{}) {
	if !verboseFlag {
		return
	}
	muConsole.Lock()
	defer muConsole.Unlock()
	fmt.Fprintf(os.Stderr, "%s %s\n", magenta("[DEBUG]"), fmt.Sprintf(format, args...))
}

// logInfo imprime mensagens informativas de progresso, respeitando -q/--quiet.
// BUGFIX (BUG-12): antes só updateProgress() respeitava --quiet; printWelcome,
// showQuickStats e showFinalSummary continuavam imprimindo mesmo em modo silencioso.
func logInfo(format string, args ...interface{}) {
	if quietFlag {
		return
	}
	fmt.Printf(format, args...)
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

	// FEATURE (--glossary): carrega o glossário de termos protegidos, se informado.
	if err := loadGlossary(glossaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível carregar o glossário")), yellow(glossaryPath), err)
		os.Exit(1)
	}
	if len(glossary) > 0 && !quietFlag {
		fmt.Printf("%s %s: %d %s\n", cyan(">>"), white(T("Glossário carregado")), len(glossary), T("termo(s)"))
	}

	// FEATURE (--dry-run): avisa o usuário que nenhuma chamada de rede/gravação real ocorrerá.
	if dryRunFlag && !quietFlag {
		fmt.Printf("%s %s\n", yellow(T("[DRY-RUN]")), white(T("Simulação ativa: nenhuma chamada de rede ou gravação real será feita.")))
	}

	loadCache()
	defer saveCache()

	// FEATURE: salva o cache automaticamente em caso de Ctrl+C (SIGINT) ou SIGTERM,
	// evitando perder as traduções já obtidas em uma execução longa interrompida.
	setupSignalHandler()

	if selfTestFlag {
		runFullSelfTest()
		return // BUGFIX: os.Exit(0) pulava o defer saveCache(); usar return preserva o cache.
	}

	if cleanCacheFlag {
		doCleanCache()
		return // BUGFIX: idem — --clean-cache não persistia a limpeza antes desta correção.
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
	atomic.StoreInt64(&fileCacheHits, 0)    // BUGFIX: reseta estatísticas por-arquivo
	atomic.StoreInt64(&fileNetCalls, 0)     // BUGFIX: idem
	atomic.StoreInt64(&fileGlossaryHits, 0) // BUGFIX: idem
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

var (
	tCache   = make(map[string]string) // BUGFIX: memoização de T()/TN() (item 10)
	tCacheMu sync.Mutex
)

// T traduz uma string da interface do programa via gettext. BUGFIX: antes disparava um
// subprocesso "gettext" a CADA chamada, mesmo repetindo o mesmo msgid entre múltiplos
// arquivos processados na mesma execução (T() é chamada ~90 vezes no código). Agora
// memoiza o resultado em memória, disparando o subprocesso no máximo uma vez por msgid.
func T(msgid string) string {
	tCacheMu.Lock()
	if v, ok := tCache[msgid]; ok {
		tCacheMu.Unlock()
		return v
	}
	tCacheMu.Unlock()

	cmd := execCommand("gettext", "-d", _APP_, msgid)
	out, err := cmd.Output()
	result := msgid
	if err == nil {
		result = strings.TrimSpace(string(out))
	}

	tCacheMu.Lock()
	tCache[msgid] = result
	tCacheMu.Unlock()
	return result
}

// TN traduz com suporte a plural via ngettext. Também memoizada (chave inclui n, já que
// o resultado pode variar conforme a contagem).
func TN(msgid, msgidPlural string, n int) string {
	key := fmt.Sprintf("%s\x00%s\x00%d", msgid, msgidPlural, n)
	tCacheMu.Lock()
	if v, ok := tCache[key]; ok {
		tCacheMu.Unlock()
		return v
	}
	tCacheMu.Unlock()

	cmd := execCommand("ngettext", "-d", _APP_, msgid, msgidPlural, fmt.Sprintf("%d", n))
	out, err := cmd.Output()
	result := msgidPlural
	if err == nil {
		result = strings.TrimSpace(string(out))
	} else if n == 1 {
		result = msgid
	}

	tCacheMu.Lock()
	tCache[key] = result
	tCacheMu.Unlock()
	return result
}

func runFullSelfTest() {
	muConsole.Lock()
	fmt.Printf("\n%s %s %s\n", cyan(">>"), white("INICIANDO TESTE DE ESTRESSE GLOBAL EXAUSTIVO"), yellow("v"+_VERSION_))
	muConsole.Unlock()

	fmt.Printf("    %s %-35s ", blue("→"), T("Dependências e Conectividade"))
	checkDependencies()
	fmt.Println(green("OK"))

	fmt.Printf("    %s %-35s ", blue("→"), T("Proteção de Variáveis ($VAR)"))
	orig := "User $USER em https://chili.com com %d"
	prot, marks := protectVariables(orig, "")
	rest := restoreVariables(prot, marks)
	if orig == rest && strings.Contains(prot, "CHILI_REF") {
		fmt.Println(green("OK"))
	} else {
		fmt.Println(red("FALHA"))
	}

	fmt.Printf("\n%s %s\n\n", green("✔"), white(T("SISTEMA 100% VALIDADO EM TODOS OS NÍVEIS.")))
}

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

func runTranslationLoop(ext, baseName string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs)
	targetBase := baseName
	if selfFlag {
		targetBase = _APP_ + ".go"
	}
	
	// Verifica se é uma extensão de manual ( .1 a .9 )
	isMan, _ := regexp.MatchString(`^\.[1-9]$`, ext)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			
			if isMan {
				translateManPage(currentFile, l)
			} else {
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
			}
			
			atomic.AddInt32(&langsDone, 1)
			done := atomic.LoadInt32(&langsDone) // BUGFIX: leitura não-atômica de langsDone (mesma classe do BUG-02, não pega na correção anterior)
			muConsole.Lock()
			if !selfTestFlag {
				fmt.Printf("\r    %s %s / %s %s", yellow(T("[STATUS]")), green(done), green(len(targetLangs)), T("idiomas concluídos..."))
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

	// FEATURE (glossário): se o texto inteiro corresponde a um termo do glossário,
	// resolve direto sem cache nem rede — glossário sempre tem prioridade.
	if fixed, ok := glossaryExactMatch(text, lang); ok {
		atomic.AddInt64(&glossaryHits, 1)     // BUGFIX: antes não entrava nas estatísticas
		atomic.AddInt64(&fileGlossaryHits, 1) // BUGFIX: idem, por-arquivo
		return fixed
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
		mu.Unlock()
		atomic.AddInt64(&cacheHits, 1)     // BUGFIX: contador atômico, fora do lock de cacheData
		atomic.AddInt64(&fileCacheHits, 1) // BUGFIX: estatística por-arquivo
		return entry.Value
	}
	mu.Unlock()

	// FEATURE (--dry-run): simula a tradução sem chamar rede nem gravar no cache.
	// BUGFIX: este check precisa vir ANTES do "if !isOnline", senão --dry-run fica
	// mudo (retorna o texto original sem simular nada) sempre que a máquina estiver
	// sem internet — que é justamente um cenário comum para querer usar --dry-run.
	// Descoberto rodando o binário de verdade num ambiente sem conectividade real.
	if dryRunFlag {
		atomic.AddInt64(&netCalls, 1)
		atomic.AddInt64(&fileNetCalls, 1)
		// BUGFIX: antes, o dry-run só testava o match EXATO do glossário; a proteção de
		// termos DENTRO de frases (protectVariables/protectGlossaryTerms) só rodava no
		// caminho real de tradução, então o usuário não conseguia validar via --dry-run
		// se um termo em meio a uma frase seria protegido corretamente. Agora aplicamos
		// a mesma proteção e mostramos o resultado (com as traduções fixas do glossário
		// já substituídas), sem chamar rede.
		protectedText, placeholders := protectVariables(text, lang)
		simulated := restoreVariables(protectedText, placeholders)
		return fmt.Sprintf("[DRY-RUN:%s] %s", lang, simulated)
	}

	if !isOnline {
		return text
	}

	transLang := strings.ReplaceAll(lang, "_", "-")
	protectedText, placeholders := protectVariables(text, lang)
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
	atomic.AddInt64(&netCalls, 1)     // BUGFIX: antes era netCalls++ sem lock/atomic (data race)
	atomic.AddInt64(&fileNetCalls, 1) // BUGFIX: estatística por-arquivo
	mu.Lock()
	cacheData[lang][normID] = CacheEntry{Value: res, LastUsed: time.Now()}
	mu.Unlock()
	return res
}

// reportCmdError centraliza o log/aviso de falhas de subprocessos externos.
// BUGFIX (BUG-08): antes esses erros eram sistematicamente descartados (`.Run()` sem
// checar retorno), fazendo etapas seguintes falharem de forma críptica e sem indicar a
// causa raiz (ex: .pot ausente porque xgettext falhou silenciosamente).
func reportCmdError(step string, err error) {
	if err == nil {
		return
	}
	logVerbose("%s: %v", step, err)
	muConsole.Lock()
	fmt.Fprintf(os.Stderr, "%s %s: %v\n", red(T("[AVISO]")), white(step), err)
	muConsole.Unlock()
}

func prepareGettext(inputPath, baseName, lang string) {
	cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	pot := filepath.Join("pot", cleanName+".pot")
	err := execCommand("xgettext", "--from-code=UTF-8", "--language="+lang, "--keyword=gettext", "--keyword=_", "--keyword=T", "--keyword=TN:1,2", "--force-po", "-o", pot, inputPath).Run()
	reportCmdError("xgettext ("+baseName+")", err)
	stampPotHeader(pot, "")
}

func prepareGettextSelf(inputPath string) {
	pot := filepath.Join("pot", _APP_+".pot")
	err := execCommand("xgettext", "--from-code=UTF-8", "--keyword=T", "--keyword=TN:1,2", "--no-wrap", "-o", pot, inputPath).Run()
	reportCmdError("xgettext --self", err)
	stampPotHeader(pot, "")
}

func writeMsgfmtToMo(base, lang string) {
	cleanBase := strings.TrimSuffix(base, filepath.Ext(base))
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	if dryRunFlag {
		// FEATURE (--dry-run): não cria diretórios nem roda msgfmt (não gera .mo real).
		logVerbose("[DRY-RUN] não geraria %s", filepath.Join(dir, cleanBase+".mo"))
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		reportCmdError("mkdir "+dir, err)
		return
	}
	poFile := filepath.Join("pot", fmt.Sprintf("%s-%s.po", cleanBase, lang))
	moFile := filepath.Join(dir, cleanBase+".mo")
	err := execCommand("msgfmt", "-f", poFile, "-o", moFile).Run()
	reportCmdError(fmt.Sprintf("msgfmt (%s, %s)", cleanBase, lang), err)
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
	pflag.BoolVar(&dryRunFlag, "dry-run", false, T("Simula a execução sem chamadas de rede nem gravação de arquivos"))
	pflag.StringVar(&glossaryPath, "glossary", "", T("Arquivo com termos que nunca devem ser traduzidos (um 'termo' ou 'termo=tradução_fixa' por linha)"))
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

	// BUGFIX (BUG-03): make(chan struct{}, jobs) entra em panic se jobs <= 0.
	// Valida e normaliza o valor de -j/--jobs.
	const maxJobs = 64
	if jobs < 1 {
		fmt.Fprintf(os.Stderr, "%s %s\n", yellow(T("[AVISO]")), white(T("--jobs deve ser >= 1; usando o padrão 8.")))
		jobs = 8
	} else if jobs > maxJobs {
		fmt.Fprintf(os.Stderr, "%s %s\n", yellow(T("[AVISO]")), white(fmt.Sprintf(T("--jobs limitado a %d."), maxJobs)))
		jobs = maxJobs
	}
}

func setupEnvironment(ext, baseName, langName string) {
	isMan, _ := regexp.MatchString(`^\.[1-9]$`, ext)

	if isMan {
		if !dryRunFlag { // BUGFIX: --dry-run não criava mais os arquivos de saída, mas ainda
			os.MkdirAll("man", 0755) // criava os diretórios e (no fluxo gettext) o .pot real
		}
		return
	}

	switch ext {
	case ".md", ".markdown":
		if !dryRunFlag {
			os.MkdirAll("doc", 0755)
		}
	case ".txt":
		if !dryRunFlag {
			os.MkdirAll("txt", 0755)
		}
	case ".json":
		if !dryRunFlag {
			os.MkdirAll("json", 0755)
		}
	case ".yaml", ".yml":
		if !dryRunFlag {
			os.MkdirAll("yml", 0755)
		}
	case ".html", ".htm":
		if !dryRunFlag {
			os.MkdirAll("html", 0755)
		}
	default:
		if dryRunFlag {
			// BUGFIX: antes, mesmo em --dry-run, esta branch rodava xgettext de verdade
			// (gravando um .pot real) e copiava arquivos .pot de entrada para pot/.
			// Agora --dry-run também pula o preparo do pipeline gettext.
			logVerbose("[DRY-RUN] preparo do pipeline gettext (.pot/xgettext) pulado para %s", baseName)
			return
		}
		os.MkdirAll("pot", 0755)
		targetPot := filepath.Join("pot", baseName)
		if ext == ".pot" {
			absInput, _ := filepath.Abs(currentFile)
			absTarget, _ := filepath.Abs(targetPot)
			if absInput != absTarget {
				if err := copyFile(currentFile, targetPot); err != nil {
					logVerbose("setupEnvironment: %v", err)
					fmt.Printf("%s %s: %v\n", red(T("ERRO:")), white(T("Falha ao copiar arquivo .pot")), err)
				}
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

// --- FUNÇÕES DE TRADUÇÃO POR FORMATO ---

func translateManPage(inputPath, lang string) {
	content, _ := os.ReadFile(inputPath)
	lines := strings.Split(string(content), "\n")
	var translatedLines []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Protege macros roff (linhas que começam com ponto)
		if strings.HasPrefix(trimmed, ".") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				macro := parts[0]
				rest := strings.Join(parts[1:], " ")
				// Traduz apenas o que vem depois da macro, se não for comentário
				if !strings.HasPrefix(rest, "\\\"") { // BUGFIX: raw string com crase (`\"`) confundia o parser C do xgettext
					translated := callUniversalTranslator(rest, lang)
					translatedLines = append(translatedLines, macro+" "+translated)
				} else {
					translatedLines = append(translatedLines, line)
				}
			} else {
				translatedLines = append(translatedLines, line)
			}
		} else if trimmed == "" {
			translatedLines = append(translatedLines, line)
		} else {
			translated := callUniversalTranslator(line, lang)
			translatedLines = append(translatedLines, translated)
		}

		if i%10 == 0 || i == len(lines)-1 {
			updateProgress(lang, i+1, len(lines), "MAN")
		}
	}
	
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outFile := filepath.Join("man", fmt.Sprintf("%s-%s%s", base, lang, ext))
	if err := dryWriteFile(outFile, []byte(strings.Join(translatedLines, "\n"))); err != nil { // BUGFIX: erro antes era ignorado
		logVerbose("gravação de %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translateHTML(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	reTag := regexp.MustCompile(`(?s)<.*?>`)

	// FEATURE: nunca traduzir o conteúdo de <script>...</script> e <style>...</style> —
	// antes, o código JS/CSS dentro desses blocos era enviado ao motor de tradução como
	// se fosse texto comum, corrompendo o arquivo.
	reScriptOpen := regexp.MustCompile(`(?i)<script[^>]*>`)
	reScriptClose := regexp.MustCompile(`(?i)</script\s*>`)
	reStyleOpen := regexp.MustCompile(`(?i)<style[^>]*>`)
	reStyleClose := regexp.MustCompile(`(?i)</style\s*>`)
	inRawBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inRawBlock && (reScriptOpen.MatchString(line) || reStyleOpen.MatchString(line)) {
			inRawBlock = true
			translatedLines = append(translatedLines, line)
			if reScriptClose.MatchString(line) || reStyleClose.MatchString(line) {
				inRawBlock = false // abre e fecha na mesma linha
			}
			continue
		}
		if inRawBlock {
			translatedLines = append(translatedLines, line)
			if reScriptClose.MatchString(line) || reStyleClose.MatchString(line) {
				inRawBlock = false
			}
			continue
		}

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
	if err := dryWriteFile(outFile, []byte(strings.Join(translatedLines, "\n"))); err != nil { // BUGFIX: erro antes era ignorado
		logVerbose("gravação de %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
	}
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

	// FEATURE: preserva bloco de front-matter YAML (delimitado por "---" logo no início
	// do arquivo), comum em geradores de site estático (Jekyll, Hugo). Antes, essas linhas
	// (title:, date:, tags: etc.) eram enviadas ao tradutor como texto comum.
	startIdx := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		translatedLines = append(translatedLines, lines[0])
		startIdx = 1
		for startIdx < len(lines) {
			translatedLines = append(translatedLines, lines[startIdx])
			if strings.TrimSpace(lines[startIdx]) == "---" {
				startIdx++
				break
			}
			startIdx++
		}
	}

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
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
	if err := dryWriteFile(outFile, []byte(strings.Join(translatedLines, "\n"))); err != nil { // BUGFIX: erro antes era ignorado
		logVerbose("gravação de %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
	}
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
		if i%10 == 0 || i == len(lines)-1 { updateProgress(lang, i+1, len(lines), "TXT") }
	}
	if err := dryWriteFile(outFile, []byte(strings.Join(translatedLines, "\n"))); err != nil { // BUGFIX: erro antes era ignorado
		logVerbose("gravação de %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

// poUnescape converte uma linha de string PO (ex: "Ola \"mundo\"\n") no texto real.
func poUnescape(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "\"")
	raw = strings.TrimSuffix(raw, "\"")
	// BUGFIX: as substituições abaixo usavam raw strings com crase (ex: `\"`), que o
	// xgettext --language=C interpretava como string C não terminada, corrompendo a
	// extração do .pot a partir daqui. Strings normais com escape têm o mesmo valor
	// em runtime e são compatíveis com o parser do xgettext.
	raw = strings.ReplaceAll(raw, "\\\"", "\x00") // marcador temporário p/ não colidir com \\ depois
	raw = strings.ReplaceAll(raw, "\\n", "\n")
	raw = strings.ReplaceAll(raw, "\\t", "\t")
	raw = strings.ReplaceAll(raw, "\\\\", "\\")
	raw = strings.ReplaceAll(raw, "\x00", "\"")
	return raw
}

// poJoinMsgid junta as linhas cruas de um msgid (possivelmente multilinha) no texto real.
func poJoinMsgid(rawLines []string) string {
	var sb strings.Builder
	for _, l := range rawLines {
		sb.WriteString(poUnescape(l))
	}
	return sb.String()
}

// poEscape converte texto real de volta para uma string PO de uma linha só (válido no formato .po).
func poEscape(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "\"", "\\\"")
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\t", "\\t")
	return text
}

func translateFile(baseName, lang string) {
	cleanBase := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", cleanBase, lang))
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", cleanBase, lang))

	if dryRunFlag {
		// BUGFIX: antes, mesmo em --dry-run, esta função tentava abrir/gravar o .po de
		// verdade (e, sem o .pot gerado por setupEnvironment em modo dry-run, isso imprimia
		// um erro confuso de "não foi possível abrir"). Agora simula sem tocar em disco.
		logVerbose("[DRY-RUN] pipeline .po simulado para %s (%s); nenhum .po/.mo real seria gerado", cleanBase, lang)
		updateProgress(lang, 1, 1, "PO")
		updateProgress(lang, 1, 1, "OK")
		return
	}

	stampPotHeader(poTmp, lang)

	file, err := os.Open(poTmp)
	if err != nil {
		// BUGFIX (BUG-09): erro de abertura não pode ser silenciado — sem isso o idioma
		// falha silenciosamente sem gerar .po nem aviso nenhum.
		logVerbose("translateFile: falha ao abrir %s: %v", poTmp, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível abrir")), yellow(poTmp), err)
		return
	}
	defer file.Close()

	output, errCreate := os.Create(poFinal)
	if errCreate != nil {
		logVerbose("translateFile: falha ao criar %s: %v", poFinal, errCreate)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível criar")), yellow(poFinal), errCreate)
		return
	}
	defer output.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// BUGFIX (BUG-06): a versão anterior excluía toda linha exatamente igual a `msgid ""`
	// para pular o cabeçalho do .po — mas msgids multilinha REAIS também começam com
	// `msgid ""` seguido de linhas de continuação, então eram silenciosamente ignorados
	// (nunca traduzidos). Agora só ignoramos o bloco de cabeçalho (antes do primeiro
	// comentário "#:", que é como o xgettext/stampPotHeader delimita onde as entradas
	// reais começam); depois disso, TODO "msgid " inicia uma entrada válida.
	// BUGFIX (item 8): antes contava linhas "msgid " para o total, mas 'current' avança
	// uma vez por linha msgstr/msgstr[N] processada — uma entrada plural tem 1 "msgid "
	// mas várias "msgstr[N]", desalinhando a barra de progresso. Agora conta msgstr(s).
	totalMsgids := 0
	pastHeaderCount := false
	for _, l := range lines {
		if strings.HasPrefix(l, "#:") {
			pastHeaderCount = true
		}
		if pastHeaderCount && (strings.HasPrefix(l, "msgstr ") || strings.HasPrefix(l, "msgstr[")) {
			totalMsgids++
		}
	}

	current := 0
	var msgidLines []string
	var pluralLines []string // BUGFIX (item 8): linhas de msgid_plural, se a entrada for plural
	collecting := ""         // "" | "msgid" | "plural" — fase de continuação em andamento
	haveEntry := false       // true entre o início de um msgid e o fim de suas msgstr(s)
	msgidPrinted := false
	pastHeader := false

	for _, line := range lines {
		if strings.HasPrefix(line, "#:") {
			pastHeader = true
		}

		switch {
		case pastHeader && strings.HasPrefix(line, "msgid_plural "):
			// BUGFIX (item 8): antes esta linha não era reconhecida (não começa com
			// "msgid " — tem "msgid_plural "), então caía no default e passava intocada,
			// e as linhas msgstr[N] associadas também nunca eram traduzidas.
			pluralLines = []string{strings.TrimPrefix(line, "msgid_plural ")}
			collecting = "plural"

		case pastHeader && strings.HasPrefix(line, "msgid "):
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
			pluralLines = nil
			collecting = "msgid"
			haveEntry = true
			msgidPrinted = false

		case pastHeader && strings.HasPrefix(line, "msgstr[") && haveEntry:
			collecting = "" // fim da fase de coleta de msgid/msgid_plural desta entrada
			current++
			updateProgress(lang, current, totalMsgids, "PO")
			if !msgidPrinted {
				fmt.Fprintf(output, "msgid %s\n", strings.Join(msgidLines, "\n"))
				if pluralLines != nil {
					fmt.Fprintf(output, "msgid_plural %s\n", strings.Join(pluralLines, "\n"))
				}
				msgidPrinted = true
			}
			idx, ok := poPluralIndex(line)
			if !ok {
				// BUGFIX: índice malformado — reproduz a linha original em vez de
				// arriscar rotular como "msgstr[0]" e duplicar uma entrada existente.
				fmt.Fprintln(output, line)
				continue
			}
			// Convenção gettext: msgstr[0] usa o msgid (singular); msgstr[1] em diante
			// usa o msgid_plural (se existir) como base para a tradução.
			source := msgidLines
			if idx > 0 && pluralLines != nil {
				source = pluralLines
			}
			original := poJoinMsgid(source)
			translated := callUniversalTranslator(original, lang)
			if translated == "" {
				translated = original
			}
			fmt.Fprintf(output, "msgstr[%d] \"%s\"\n", idx, poEscape(translated))

		case pastHeader && strings.HasPrefix(line, "msgstr ") && haveEntry:
			current++
			updateProgress(lang, current, totalMsgids, "PO")
			original := poJoinMsgid(msgidLines)
			translated := callUniversalTranslator(original, lang)
			if translated == "" {
				translated = original
			}
			fmt.Fprintf(output, "msgid %s\nmsgstr \"%s\"\n", strings.Join(msgidLines, "\n"), poEscape(translated))
			haveEntry = false
			collecting = ""

		case collecting == "plural":
			// linhas de continuação do msgid_plural multilinha
			pluralLines = append(pluralLines, line)

		case collecting == "msgid":
			// linhas de continuação do msgid multilinha (`"parte 2"`, etc.)
			msgidLines = append(msgidLines, line)

		default:
			fmt.Fprintln(output, line)
		}
	}
	os.Remove(poTmp)
	updateProgress(lang, totalMsgids, totalMsgids, "OK")
}

// poPluralIndex extrai o índice N de uma linha "msgstr[N] ...". O segundo retorno indica
// se o parse teve sucesso — BUGFIX: antes retornava 0 tanto para "msgstr[0]" legítimo
// quanto para um índice malformado, arriscando gerar um "msgstr[0]" duplicado no .po de
// saída; agora o chamador pode tratar a falha de forma defensiva.
func poPluralIndex(line string) (int, bool) {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || end < start {
		return 0, false
	}
	idx, err := strconv.Atoi(strings.TrimSpace(line[start+1 : end]))
	if err != nil {
		return 0, false
	}
	return idx, true
}

func translateJSON(path, lang string) {
	ext := strings.ToLower(filepath.Ext(path))
	isYAML := ext == ".yaml" || ext == ".yml"
	targetDir := "json"
	if isYAML {
		targetDir = "yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logVerbose("translateJSON: erro lendo %s: %v", path, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível ler")), yellow(path), err)
		return
	}

	var obj interface{}
	if isYAML {
		// BUGFIX (BUG-04): antes usava encoding/json também para .yaml/.yml, o que só
		// funcionava para YAML compatível com sintaxe JSON (flow style). YAML "de verdade"
		// (block style, comentários, etc.) falhava com erro ignorado, gerando saída vazia.
		if err := yaml.Unmarshal(data, &obj); err != nil {
			logVerbose("translateJSON: YAML inválido em %s: %v", path, err)
			fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("YAML inválido em")), yellow(path), err)
			return
		}
	} else {
		if err := json.Unmarshal(data, &obj); err != nil {
			logVerbose("translateJSON: JSON inválido em %s: %v", path, err)
			fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("JSON inválido em")), yellow(path), err)
			return
		}
	}

	obj = translateValue(obj, lang)

	var out []byte
	if isYAML {
		out, err = yaml.Marshal(obj)
	} else {
		out, err = json.MarshalIndent(obj, "", "  ")
	}
	if err != nil {
		logVerbose("translateJSON: erro serializando saída de %s: %v", path, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Falha ao gerar saída para")), yellow(path), err)
		return
	}

	outFile := filepath.Join(targetDir, fmt.Sprintf("%s-%s%s", strings.TrimSuffix(filepath.Base(path), ext), lang, ext))
	if err := dryWriteFile(outFile, out); err != nil { // FEATURE (--dry-run)
		logVerbose("translateJSON: erro gravando %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
		return
	}
	updateProgress(lang, 100, 100, strings.ToUpper(targetDir))
}

// translateValue percorre recursivamente mapas, arrays e strings, traduzindo cada string
// encontrada. BUGFIX (BUG-05): a antiga translateMap só recursionava em
// map[string]interface{}, deixando strings dentro de arrays ([]interface{}) — comuns em
// arquivos de i18n — sem tradução.
func translateValue(v interface{}, lang string) interface{} {
	switch val := v.(type) {
	case string:
		return callUniversalTranslator(val, lang)
	case map[string]interface{}:
		for k, vv := range val {
			val[k] = translateValue(vv, lang)
		}
		return val
	case []interface{}:
		for i, vv := range val {
			val[i] = translateValue(vv, lang)
		}
		return val
	default:
		return v
	}
}

// --- UTILITÁRIOS ---

func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag || total == 0 {
		return
	}
	muConsole.Lock()
	defer muConsole.Unlock()
	pos := langPositions[lang]
	// NOTA: pos só é 0 se 'lang' não estiver em langPositions (map lookup retorna o
	// zero-value em Go). Em uso normal isso não acontece, pois updateProgress só é
	// chamado com idiomas vindos de targetLangs — mas o guard evita imprimir códigos
	// de escape ANSI errados caso um lang desconhecido chegue aqui no futuro.
	if pos == 0 {
		return
	}
	percent := (current * 100) / total
	width := 40
	filled := (percent * width) / 100
	bar := blue(strings.Repeat("░", filled)) + strings.Repeat(" ", width-filled)
	langStr := fmt.Sprintf("%-7s", lang)
	fmt.Printf("\033[%dA\r\033[K    → %s %s [%s] %3d%% %-5s\033[%dB", pos, blue("→"), cyan(langStr), bar, percent, cyan(suffix), pos)
}

func prepareMsginit(base, lang string) {
	if dryRunFlag {
		// BUGFIX: antes, esta função rodava msginit normalmente mesmo em --dry-run;
		// agora o .pot nem é gerado (ver setupEnvironment), então nem tenta.
		logVerbose("[DRY-RUN] msginit simulado para %s (%s)", base, lang)
		return
	}
	cleanBase := strings.TrimSuffix(base, filepath.Ext(base))
	pot := filepath.Join("pot", cleanBase+".pot")
	po := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", cleanBase, lang))
	os.Remove(po)
	if _, err := os.Stat(pot); err != nil {
		// BUGFIX (BUG-08): antes seguia direto para msginit mesmo sem o .pot existir
		// (caso xgettext tivesse falhado silenciosamente antes), gerando erro críptico.
		reportCmdError(fmt.Sprintf("msginit (%s, %s): .pot ausente", cleanBase, lang), err)
		return
	}
	err := execCommand("msginit", "--no-translator", "-l", lang, "-i", pot, "-o", po).Run()
	reportCmdError(fmt.Sprintf("msginit (%s, %s)", cleanBase, lang), err)
}

// --- GLOSSÁRIO (FEATURE) ---
// Formato do arquivo (uma entrada por linha, "#" inicia comentário):
//   termo                              -> nunca traduz, mantém o termo tal como escrito
//   termo=tradução_fixa                -> sempre substitui por "tradução_fixa", em qualquer idioma-alvo
//   termo=en:Product;fr:Produit        -> tradução fixa DIFERENTE por idioma-alvo (BUGFIX)
//                                          (se o idioma-alvo não tiver entrada, mantém o termo original)

type glossaryEntry struct {
	term    string            // termo original, como aparece no arquivo de glossário
	fixed   string            // tradução fixa global; vazio significa "preservar o termo original"
	perLang map[string]string // BUGFIX: tradução fixa por idioma-alvo ("en:Product;fr:Produit")
}

type glossaryRule struct {
	re    *regexp.Regexp
	entry glossaryEntry
}

func loadGlossary(path string) error {
	glossary = make(map[string]glossaryEntry)
	glossaryRules = nil
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		term := strings.TrimSpace(parts[0])
		if term == "" {
			continue
		}
		fixed := ""
		var perLang map[string]string
		if len(parts) == 2 {
			val := strings.TrimSpace(parts[1])
			// BUGFIX: antes "termo=tradução" era sempre global (mesma tradução para
			// TODOS os idiomas-alvo). Agora também aceita "termo=idioma:trad;idioma2:trad2"
			// para traduções fixas específicas por idioma.
			if strings.Contains(val, ":") {
				perLang = make(map[string]string)
				for _, pair := range strings.Split(val, ";") {
					kv := strings.SplitN(pair, ":", 2)
					if len(kv) != 2 {
						continue
					}
					lc := strings.ToLower(strings.TrimSpace(kv[0]))
					tv := strings.TrimSpace(kv[1])
					if lc != "" {
						perLang[lc] = tv
					}
				}
				if len(perLang) == 0 {
					fixed = val // nenhum par válido: trata como tradução fixa global
					perLang = nil
				}
			} else {
				fixed = val
			}
		}
		entry := glossaryEntry{term: term, fixed: fixed, perLang: perLang}
		glossary[strings.ToLower(term)] = entry
		// BUGFIX: `\b` do RE2 só reconhece [0-9A-Za-z_] como "caractere de palavra",
		// então termos que começam/terminam com letra acentuada (café, ação, não)
		// podiam falhar silenciosamente. Capturamos a fronteira esquerda no grupo 1 e
		// checamos a fronteira direita manualmente (via runas Unicode) em applyGlossaryRule,
		// sem consumir o caractere seguinte no match.
		re, errRe := regexp.Compile(`(?i)(^|[^\p{L}\p{N}_])(` + regexp.QuoteMeta(term) + `)`)
		if errRe != nil {
			logVerbose("loadGlossary: termo ignorado (regex inválida) %q: %v", term, errRe)
			continue
		}
		glossaryRules = append(glossaryRules, glossaryRule{re: re, entry: entry})
	}
	// BUGFIX: sem isso, um termo curto listado antes de um termo mais longo que o contém
	// (ex: "Paulo" antes de "São Paulo") "engolia" a ocorrência primeiro, e a regra do
	// termo mais específico deixava de casar (o texto já não tinha mais "São Paulo"
	// literal). Ordenar do termo mais longo para o mais curto resolve a maioria dos casos
	// de sobreposição, priorizando sempre o match mais específico.
	sort.Slice(glossaryRules, func(i, j int) bool {
		return len(glossaryRules[i].entry.term) > len(glossaryRules[j].entry.term)
	})
	return scanner.Err()
}

// resolveGlossaryTranslation resolve a tradução fixa de uma entrada do glossário para um
// idioma-alvo específico. BUGFIX: antes "termo=tradução" era global; agora prioriza uma
// tradução específica de idioma, se houver, com fallback para o prefixo do idioma
// (ex: "pt" cobre "pt_BR") e depois para a tradução global. Retorna "" se não houver
// nenhuma tradução fixa aplicável (chamador deve então preservar o termo original).
func resolveGlossaryTranslation(entry glossaryEntry, lang string) string {
	if entry.perLang != nil {
		normLang := strings.ToLower(strings.ReplaceAll(lang, "-", "_"))
		if v, ok := entry.perLang[normLang]; ok {
			return v
		}
		if idx := strings.IndexAny(normLang, "_-"); idx > 0 {
			if v, ok := entry.perLang[normLang[:idx]]; ok {
				return v
			}
		}
	}
	return entry.fixed
}

// glossaryExactMatch resolve o caso em que o TEXTO INTEIRO enviado a traduzir é
// exatamente um termo do glossário (comum em valores de JSON/YAML e msgids curtos).
func glossaryExactMatch(text, lang string) (string, bool) {
	if len(glossary) == 0 {
		return "", false
	}
	entry, ok := glossary[strings.ToLower(strings.TrimSpace(text))]
	if !ok {
		return "", false
	}
	if v := resolveGlossaryTranslation(entry, lang); v != "" {
		return v, true
	}
	return entry.term, true
}

// isWordRune define o que conta como "caractere de palavra" para fins de fronteira,
// usando classes Unicode (letras/dígitos de qualquer idioma), diferente do `\b` do RE2
// que só reconhece ASCII e falha com termos acentuados.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// protectGlossaryTerms protege ocorrências de termos do glossário DENTRO de um texto
// maior (ex: um termo de produto no meio de uma frase de documentação), usando o
// mesmo mecanismo de placeholders do protectVariables.
func protectGlossaryTerms(text, lang string, placeholders map[string]string) string {
	if len(glossaryRules) == 0 {
		return text
	}
	idx := 0
	for _, rule := range glossaryRules {
		text = applyGlossaryRule(rule, text, lang, placeholders, &idx)
	}
	return text
}

// applyGlossaryRule aplica uma regra de glossário a todo o texto, validando a fronteira
// direita manualmente (sem consumir o caractere seguinte no match), o que corrige o
// problema do `\b` com termos que terminam em letra acentuada.
func applyGlossaryRule(rule glossaryRule, text, lang string, placeholders map[string]string, idx *int) string {
	matches := rule.re.FindAllStringSubmatchIndex(text, -1)
	if matches == nil {
		return text
	}
	var sb strings.Builder
	last := 0
	for _, m := range matches {
		// m[0]:m[1] = match inteiro (fronteira esquerda + termo); m[4]:m[5] = só o termo
		termStart, termEnd := m[4], m[5]
		if termStart < last {
			continue // já coberto por um match anterior nesta mesma passada
		}
		if termEnd < len(text) {
			r, _ := utf8.DecodeRuneInString(text[termEnd:])
			if isWordRune(r) {
				continue // caractere seguinte é de palavra: não é uma fronteira real
			}
		}
		sb.WriteString(text[last:termStart])
		p := fmt.Sprintf("CHILI_GLOSS_%d_CHILI", *idx)
		*idx++
		replacement := resolveGlossaryTranslation(rule.entry, lang)
		if replacement == "" {
			replacement = text[termStart:termEnd] // preserva a capitalização original
		}
		placeholders[p] = replacement
		sb.WriteString(p)
		last = termEnd
	}
	sb.WriteString(text[last:])
	return sb.String()
}

func protectVariables(text, lang string) (string, map[string]string) {
	// BUGFIX (BUG-10): `%[a-z]` só cobria especificadores simples (%s, %d) e deixava passar
	// variantes com flags/largura/precisão (%.2f, %5d, %-10s) e maiúsculas (%S) sem proteção.
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[-+ 0#]*[0-9]*(?:\.[0-9]+)?[a-zA-Z]|` + "`[^`\\n]+`" + `|!\[.*?\]\(.*?\)|\[.*?\]\(.*?\)|https?://[^\s]+)`)
	placeholders := make(map[string]string)
	// FEATURE (glossário): protege termos do glossário antes de qualquer outra proteção,
	// para que nunca sejam enviados ao motor de tradução.
	protected := protectGlossaryTerms(text, lang, placeholders)
	matches := re.FindAllString(protected, -1)
	for i, match := range matches {
		p := fmt.Sprintf("CHILI_REF_%d_CHILI", i)
		placeholders[p] = match
		protected = strings.Replace(protected, match, p, 1)
	}
	return protected, placeholders
}

func restoreVariables(text string, p map[string]string) string {
	for k, v := range p { text = strings.Replace(text, k, v, -1) }
	return text
}

func detectFileType(path string) (ext string, lang string, desc string) {
	ext = strings.ToLower(filepath.Ext(path))
	
	// Caso 1: Arquivo sem extensão
	if ext == "" {
		detected, _ := getShebangInfo(path)
		if detected != "" {
			return "", detected, fmt.Sprintf(T("Script (%s)"), green(detected))
		}
		return ".txt", "text", T("Texto Simples (sem extensão)")
	}

	// Caso 2: Man Pages ( .1 a .9 )
	isMan, _ := regexp.MatchString(`^\.[1-9]$`, ext)
	if isMan {
		return ext, "manpage", fmt.Sprintf(T("Manual do Linux (%s)"), cyan(ext))
	}

	extMap := map[string]string{
		".sh": "shell", ".py": "python", ".php": "php", ".c": "c",
		".cpp": "c++", ".go": "go", ".pl": "perl", ".rb": "ruby",
		".html": "html", ".htm": "html",
	}

	if l, ok := extMap[ext]; ok {
		return ext, l, fmt.Sprintf(T("Código %s (%s)"), ext, green(l))
	}

	switch ext {
	case ".md", ".markdown": return ext, "markdown", T("Markdown")
	case ".txt": return ext, "text", T("Texto Simples")
	case ".json": return ext, "json", T("JSON")
	case ".yaml", ".yml": return ext, "yaml", T("YAML")
	case ".pot": return ext, "gettext", T("Template POT")
	}

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
			case strings.Contains(lower, "python"): return "python", line
			case strings.Contains(lower, "php"): return "php", line
			case strings.Contains(lower, "perl"): return "perl", line
			case strings.Contains(lower, "ruby"): return "ruby", line
			case strings.Contains(lower, "node"): return "javascript", line
			case strings.Contains(lower, "bash") || strings.Contains(lower, "sh"): return "shell", line
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

// setupSignalHandler garante que o cache seja salvo em disco caso o usuário interrompa
// a execução (Ctrl+C / SIGINT) ou o processo receba SIGTERM. Sem isso, uma execução
// longa interrompida no meio perderia todas as traduções já obtidas naquela sessão
// (o defer saveCache() de main() não é executado quando o processo é encerrado por sinal).
func setupSignalHandler() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		muConsole.Lock()
		fmt.Fprintf(os.Stderr, "\n%s %s (%v)\n", yellow(T("[AVISO]")), white(T("Interrompido — salvando cache antes de encerrar...")), sig)
		muConsole.Unlock()
		saveCache()
		os.Exit(130)
	}()
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
	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("abrir origem %q: %w", src, err)
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("criar destino %q: %w", dst, err)
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	if err != nil {
		return fmt.Errorf("copiar %q para %q: %w", src, dst, err)
	}
	return nil
}

func hasActualContent(ext, baseName string) bool {
	if selfFlag { return true }
	isMan, _ := regexp.MatchString(`^\.[1-9]$`, ext)
	if isMan { return true }
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
	if quietFlag { // BUGFIX (BUG-12): --quiet antes só afetava updateProgress()
		return
	}
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
	if quietFlag { // BUGFIX (BUG-12)
		return
	}
	// BUGFIX (BUG-07): usar contadores por-arquivo, não os totais globais acumulados
	// de todos os arquivos já processados na mesma execução.
	hits := atomic.LoadInt64(&fileCacheHits)
	netVal := atomic.LoadInt64(&fileNetCalls) // BUGFIX: antes se chamava 'net', sombreando o pacote "net" importado
	gloss := atomic.LoadInt64(&fileGlossaryHits)
	total := hits + netVal + gloss // BUGFIX: antes não incluía acertos de glossário no total
	pCache, pNet, pGloss := 0.0, 0.0, 0.0
	if total > 0 {
		pCache = (float64(hits) / float64(total)) * 100
		pNet = (float64(netVal) / float64(total)) * 100
		pGloss = (float64(gloss) / float64(total)) * 100
	}
	fmt.Printf("\n\n%s %s em %v | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n",
		green("✔"), white(T("Concluído")), time.Since(start).Round(time.Second),
		blue(T("Cache:")), hits, pCache,
		yellow(T("Net:")), netVal, pNet,
		magenta(T("Glossário:")), gloss, pGloss,
		white(T("Total:")), total)
}

func showFinalSummary(start time.Time) {
	if quietFlag { // BUGFIX (BUG-12)
		return
	}
	fmt.Printf("%s\n %s\n", white(strings.Repeat("-", 60)), yellow(T("RESUMO EXECUTIVO FINAL:")))
	fmt.Printf("    → %-15s: %v\n", T("Tempo Total"), time.Since(start).Round(time.Second))
	fmt.Printf("    → %-15s: %d\n", T("Cache Hits"), atomic.LoadInt64(&cacheHits))
	fmt.Printf("    → %-15s: %d\n", T("Chamadas Rede"), atomic.LoadInt64(&netCalls))
	if g := atomic.LoadInt64(&glossaryHits); g > 0 {
		fmt.Printf("    → %-15s: %d\n", T("Acertos Glossário"), g)
	}
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
		{"-i", "--inputfile", T("Arquivo fonte (.sh, .py, .md, .txt, .json, .yaml, .html, .pot, .[1-9])")},
		{"-l", "--language", fmt.Sprintf(T("Idiomas (ex: pt_BR,en) ou 'all' (padrão: %s)"), defLangs)},
		{"-e", "--engine", T("Motor: google, bing, yandex (padrão: google)")},
		{"-j", "--jobs", T("Traduções simultâneas (padrão: 8)")},
		{"-s", "--source", T("Idioma de origem (ex: pt, en) (padrão: auto)")},
		{"-f", "--force", T("Força nova tradução (ignora cache)")},
		{"", "--self", T("Extração especializada para o próprio chili-tradutor-go")},
		{"", "--self-test", T("Executa auto-teste de integridade")},
		{"", "--clean-cache", T("Remove entradas de cache não usadas há 30 dias")},
		{"", "--dry-run", T("Simula a execução sem chamadas de rede nem gravação de arquivos")},
		{"", "--glossary", T("Arquivo de termos protegidos (termo | termo=tradução | termo=en:x;fr:y)")},
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
