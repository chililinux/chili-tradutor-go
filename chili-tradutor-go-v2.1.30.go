/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/chililinux/chili-tradutor-go

    Updated:   seg 28 set 2026 (continuação do msgstr antigo colada na tradução — ver Changelog 2.1.30)
    Version:   2.1.30

    Changelog 2.1.30:
      - translateFile: linhas de continuação do msgstr original eram mantidas depois do
        msgstr traduzido, resultando em msgstr = tradução + texto antigo. Acontecia
        sempre em -l en (o msginit já preenche msgstr com o msgid para inglês) com
        mensagens longas, e em qualquer .po/.pot que já trouxesse msgstr multilinha;
        o msgfmt -c rejeitava o catálogo. Achado ao testar --self.

    Changelog 2.1.29 (varredura completa pós-2.1.28):
      - T()/TN(): o gettext rodava com LC_ALL=C, que faz o GNU gettext ignorar LANGUAGE
        e devolver sempre o original — a interface do programa nunca era traduzida;
        agora roda com o locale do usuário, usa "--" antes do msgid (msgids que
        começam com "-" eram lidos como opção) e não apara mais espaços do msgid
      - stampPotHeader: Plural-Forms fixo "nplurals=2; plural=(n > 1)" em TODOS os
        idiomas (errado para ru, ja, ko, zh, en...); agora preserva o Plural-Forms que
        o msginit gera para cada idioma; Language-Team deixou de ser "Portuguese" fixo;
        Project-Id-Version usa o nome do domínio traduzido, não o do chili-tradutor-go;
        .pot sai com "Language: " vazio (padrão gettext) em vez de "none"
      - stampPotHeader: comentários "#." antes do primeiro "#:" eram descartados
      - .pot/.po sem comentários "#:" (xgettext --no-location, feito à mão) não tinham
        nada traduzido; o fim do cabeçalho agora é detectado pela entrada msgid ""
      - proteção de %: "100% de" virava "100CHILI_REF_0_CHILIe" (espaço aceito como
        flag); agora exige conversão printf válida e aceita %1$s e %%
      - glossário dentro de crases/links/URLs gerava placeholder dentro de placeholder
        e a restauração (ordem aleatória de map) deixava CHILI_GLOSS_N_CHILI no
        resultado; código/URL é protegido antes do glossário e a restauração é em ordem
      - espaços iniciais/finais eram removidos (indentação, "\n" final de msgid, quebra
        de linha com dois espaços no Markdown); agora são preservados
      - hasActualContent/cleanupEmpty procuravam pot/<nome>.<ext>.pot (nunca existia);
        e qualquer .pot contava como "com conteúdo" por causa do msgid "" do cabeçalho
      - cache: chave era o texto em minúsculas ("Save" e "save" dividiam a tradução);
        agora respeita maiúsculas, separa por motor/origem quando não são os padrões
        (google/auto) e muda quando o glossário aplicado ao texto muda
      - cache gravado de forma atômica (temporário + rename); cache.json corrompido
        é guardado como cache.json.bad com aviso, em vez de sumir calado
      - man pages: só traduz argumentos de .SH, .SS, .B, .I, .SM e .SB; demais macros,
        comentários .\" e blocos .nf/.fi e .EX/.EE ficam intactos; escapes roff
        (\fB, \-, \(xx...) protegidos; espaçamento original preservado
      - --quiet ainda imprimia [STEP 2], lista de idiomas, [STATUS] e aviso final
      - checkDependencies movido para depois do parse: -V, --help e --clean-cache não
        exigem mais trans/gettext; --dry-run também não
      - glossário: "site=https://..." ou "Nota=Obs: x" viravam traduções por idioma;
        agora só é por idioma quando todas as chaves são códigos de idioma suportados;
        chaves normalizadas (zh-TW casa com zh_TW)
      - JSON: ordem das chaves preservada e números mantidos exatamente como no
        original (inteiros grandes perdiam precisão); sem escape de <, > e &
      - YAML: ordem das chaves e comentários preservados (yaml.Node)
      - JSON/YAML: não traduz valores que parecem identificadores, caminhos, URLs,
        e-mails, nomes de arquivo ou números
      - erros de leitura em man/md/txt eram ignorados e geravam saída vazia
      - código de saída: 1 se houve erro (arquivo ausente, leitura/gravação), 2 se
        terminou mas ficaram textos sem tradução, 0 se tudo ok
      - detectDistro reconhece ID_LIKE (VoidBR e derivadas do Void/Arch/Debian)
      - "Concluído em" passa por T() (o "em" estava fixo no código)

    Changelog 2.1.28:
      - checkInternet() rodava antes de parseFlags(): -V/--version, -h/--help,
        --clean-cache, --self-test e a execução sem argumentos esperavam até ~6s
        quando offline (3 destinos x 2s); agora a conexão só é testada quando há
        arquivos a processar e não é --dry-run
      - aviso de "sem conexão" movido junto com a checagem (antes aparecia até
        em --clean-cache, que não usa rede)

    Changelog 2.1.27:
      - aviso de rate limiting agora também aparece no resumo (por arquivo e final);
        o impresso durante o processamento era sobrescrito pelas barras de progresso

    Changelog 2.1.26 (erros do trans deixam de ser silenciosos):
      - stderr do trans agora é capturado (antes era descartado); o trans sai com
        código 0 mesmo quando falha e só explica o motivo no stderr
      - rate limiting detectado ("rate limit" no stderr): avisa uma vez e para de
        chamar o motor pelo resto da execução, em vez de insistir em centenas de
        pedidos que só prolongam o bloqueio
      - saída vazia do trans passa a ser repetida (antes aceitava na 1ª tentativa)
      - espera entre tentativas crescente (1s, 2s) e sem espera após a última
      - resumos mostram o último erro do trans quando há textos não traduzidos

    Changelog 2.1.25 (textos sem tradução não são mais mascarados):
      - .po: quando o trans falhava ou estava offline, o msgstr recebia uma cópia do
        msgid — o catálogo parecia traduzido, mas não estava; agora fica msgstr ""
        (padrão gettext para pendente) e é traduzido na próxima execução
      - callTranslatorStrict(): nova função que informa se traduziu; callUniversalTranslator
        passa a usá-la e mantém o fallback para o original (md, man, json, yaml, html, txt)
      - resposta vazia do trans passa a contar como falha
      - checkInternet: testava só 8.8.8.8:53 (bloqueado em muitas redes); agora tenta
        8.8.8.8:53, 1.1.1.1:53 e translate.googleapis.com:443
      - aviso em vermelho ao iniciar sem conexão (antes seguia calado)
      - resumos por arquivo e final mostram "Não traduzidos: N"

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
	"bytes" // FEATURE: captura do stderr do trans
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
	_VERSION_ = "2.1.30-20260928"
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
	hadErrors      int32 // BUGFIX: código de saída — 1 quando houve erro de arquivo/leitura/gravação
	cacheHits      int64 // BUGFIX: era int com netCalls++ fora de lock (data race); agora atômico.
	netCalls       int64 // BUGFIX: idem.
	fileCacheHits  int64 // BUGFIX: contador por-arquivo, resetado a cada processSingleFile.
	fileNetCalls   int64 // BUGFIX: idem — antes showQuickStats exibia totais acumulados de todos os arquivos.
	glossaryHits   int64 // BUGFIX: contador de acertos de glossário, antes não entrava nas estatísticas.
	fileGlossaryHits int64 // BUGFIX: idem, por-arquivo.
	failedCalls    int32
	untranslated   int32 // textos que ficaram sem tradução (offline ou falha)
	fileUntranslated int32 // idem, por-arquivo (exibido no resumo rápido)
	rateLimited      int32 // FEATURE: 1 quando o motor bloqueou por excesso de pedidos
	rateLimitOnce    sync.Once
	lastTransErr     atomic.Value // FEATURE: última mensagem de erro do trans (string)
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
	os.Exit(run())
}

// run contém o fluxo principal e devolve o código de saída. BUGFIX: antes o programa
// saía sempre com 0, mesmo com arquivo não encontrado ou textos sem tradução; e um
// os.Exit() direto em main() pularia o defer saveCache().
//   0 = tudo ok | 1 = houve erro (arquivo ausente, leitura/gravação) | 2 = ficaram textos sem tradução
func run() int {
	// BUGFIX: checkInternet() e checkDependencies() rodavam aqui, antes de parseFlags():
	// -V, -h e --clean-cache esperavam pela rede e exigiam trans/gettext instalados.
	// Agora ambos rodam só quando há arquivos a processar.
	parseFlags()

	if versionFlag {
		showVersion()
		return 0
	}

	// FEATURE (--glossary): carrega o glossário de termos protegidos, se informado.
	if err := loadGlossary(glossaryPath); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível carregar o glossário")), yellow(glossaryPath), err)
		return 1
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
		return 0 // BUGFIX: os.Exit(0) pulava o defer saveCache(); usar return preserva o cache.
	}

	if cleanCacheFlag {
		doCleanCache()
		return 0 // BUGFIX: idem — --clean-cache não persistia a limpeza antes desta correção.
	}

	allFiles := append(inputFiles, pflag.Args()...)
	if len(allFiles) == 0 {
		usage()
		return 1
	}

	// BUGFIX: a conexão só é testada aqui, quando de fato há arquivos a traduzir.
	// Em --dry-run não há chamada de rede nem ferramentas externas, então o teste de
	// conexão, o aviso e a checagem de dependências são pulados.
	if !dryRunFlag {
		checkDependencies()
		isOnline = checkInternet()
		// FEATURE: avisa quando está offline — antes seguia calado e os textos
		// saíam sem tradução.
		if !isOnline {
			fmt.Fprintf(os.Stderr, "%s %s\n", red(T("AVISO:")), white(T("sem conexão com a internet: nenhum texto será traduzido (só cache e glossário).")))
		}
	}

	startGlobal := time.Now()
	for _, file := range allFiles {
		processSingleFile(file)
	}

	if len(allFiles) > 1 {
		logInfo("\n%s %s\n", green("✔"), white(T("Todos os arquivos foram processados!"))) // BUGFIX: respeita --quiet
		showFinalSummary(startGlobal)
	}

	switch {
	case atomic.LoadInt32(&hadErrors) > 0:
		return 1
	case atomic.LoadInt32(&untranslated) > 0:
		return 2
	}
	return 0
}

// markError registra que houve um erro, para o código de saída final.
func markError() { atomic.StoreInt32(&hadErrors, 1) }

func processSingleFile(path string) {
	if _, err := os.Stat(path); err != nil {
		// BUGFIX: antes só tratava "não existe"; outros erros (permissão) seguiam adiante.
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%v)\n", red(T("ERRO:")), white(T("Arquivo não encontrado:")), yellow(path), err)
		markError()
		return
	}

	currentFile = path
	langsDone = 0
	atomic.StoreInt64(&fileCacheHits, 0)    // BUGFIX: reseta estatísticas por-arquivo
	atomic.StoreInt64(&fileNetCalls, 0)     // BUGFIX: idem
	atomic.StoreInt64(&fileGlossaryHits, 0) // BUGFIX: idem
	atomic.StoreInt32(&fileUntranslated, 0)  // FEATURE: pendências por-arquivo
	ext, langName, desc := detectFileType(path)
	baseName := filepath.Base(path)
	setupEnvironment(ext, baseName, langName)

	printWelcome(desc)
	start := time.Now()

	if !hasActualContent(ext, baseName) {
		logInfo("%s %s\n", yellow(T("[AVISO]")), white(T("Nada para traduzir ou arquivo protegido.")))
		cleanupEmpty(ext, baseName)
	} else {
		// BUGFIX: --quiet não suprimia [STEP 2] nem a lista de idiomas.
		logInfo("%s %s\n\n", yellow(T("[STEP 2]")), white(T("Iniciando processamento paralelo...")))
		for _, lang := range targetLangs {
			logInfo("    → %s %s\n", cyan(fmt.Sprintf("%-7s", lang)), yellow(T("[Aguardando...]")))
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

	// BUGFIX: antes usava execCommand (LC_ALL=C). No locale "C" o GNU gettext ignora
	// LANGUAGE e devolve sempre o msgid — a interface nunca era traduzida. Agora roda
	// com o ambiente do usuário. "--" impede que msgids iniciados com "-" (ex:
	// "--jobs deve ser >= 1") sejam lidos como opção, e a saída não é mais aparada
	// (TrimSpace cortava espaços intencionais, ex: "Dependências ausentes: ").
	cmd := exec.Command("gettext", "-d", _APP_, "--", msgid)
	out, err := cmd.Output()
	result := msgid
	if err == nil {
		result = string(out)
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

	cmd := exec.Command("ngettext", "-d", _APP_, "--", msgid, msgidPlural, fmt.Sprintf("%d", n)) // BUGFIX: idem T()
	out, err := cmd.Output()
	result := msgidPlural
	if err == nil {
		result = string(out)
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

// poHeaderEnd localiza o fim do cabeçalho de um .po/.pot (a entrada msgid "" seguida
// direto de msgstr). Retorna o índice da primeira linha do corpo (entradas reais), a
// linha crua de Plural-Forms do cabeçalho (se houver) e se havia cabeçalho.
// BUGFIX: antes o corpo era identificado pela primeira linha "#:"; arquivos sem
// comentários de referência (xgettext --no-location, .pot feito à mão) ficavam sem
// nenhuma entrada traduzida, e comentários "#." antes do primeiro "#:" eram perdidos.
func poHeaderEnd(lines []string) (bodyStart int, pluralLine string, hasHeader bool) {
	i := 0
	for i < len(lines) && !strings.HasPrefix(lines[i], "msgid ") {
		i++
	}
	if i < len(lines) && strings.TrimSpace(lines[i]) == "msgid \"\"" &&
		i+1 < len(lines) && strings.HasPrefix(lines[i+1], "msgstr ") {
		j := i + 2
		for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "\"") {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "\"Plural-Forms:") {
				// BUGFIX: o msginit quebra Plural-Forms longos (ex: ru) em várias linhas;
				// o campo só termina na linha que acaba em \n"
				k := j
				for k < len(lines)-1 && !strings.HasSuffix(strings.TrimSpace(lines[k]), "\\n\"") &&
					strings.HasPrefix(strings.TrimSpace(lines[k+1]), "\"") {
					k++
				}
				var parts []string
				for _, l := range lines[j : k+1] {
					parts = append(parts, strings.TrimSpace(l))
				}
				pluralLine = strings.Join(parts, "\n")
				j = k
			}
			j++
		}
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		return j, pluralLine, true
	}
	// sem cabeçalho: o corpo começa nos comentários que precedem a primeira entrada
	k := i
	for k > 0 && strings.HasPrefix(lines[k-1], "#") {
		k--
	}
	return k, "", false
}

// stampPotHeader troca o cabeçalho de um .pot/.po pelo cabeçalho padrão do projeto.
// domain é o domínio gettext traduzido (nome do .pot/.mo); lang é "" para o .pot.
func stampPotHeader(path, domain, lang string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")
	bodyStart, pluralLine, _ := poHeaderEnd(lines)

	// BUGFIX: antes o Plural-Forms era fixo "nplurals=2; plural=(n > 1);" para TODOS os
	// idiomas, sobrescrevendo o valor correto que o msginit gera (ru tem 3 formas,
	// ja/ko/zh têm 1, en usa n != 1). Agora o valor existente é preservado.
	if pluralLine == "" {
		if lang == "" {
			pluralLine = "\"Plural-Forms: nplurals=INTEGER; plural=EXPRESSION;\\n\""
		} else {
			pluralLine = "\"Plural-Forms: nplurals=2; plural=(n != 1);\\n\""
		}
	}
	// BUGFIX: Language-Team era "Portuguese" fixo; o .pot usava "Language: none".
	team := "none"
	if lang != "" {
		team = lang + " <https://github.com/chililinux/chili-tradutor-go>"
	}
	now := time.Now().Format("2006-01-02 15:04-0700")
	// BUGFIX: Project-Id-Version e a linha de licença citavam o chili-tradutor-go,
	// não o domínio que está sendo traduzido.
	header := fmt.Sprintf(
		"# Chili Tradutor Go - %s\n"+
			"# Copyright (C) 2019-2026 Vilmar Catafesta <vcatafesta@gmail.com>\n"+
			"# This file is distributed under the same license as the %s package.\n"+
			"msgid \"\"\n"+
			"msgstr \"\"\n"+
			"\"Project-Id-Version: %s\\n\"\n"+
			"\"POT-Creation-Date: %s\\n\"\n"+
			"\"PO-Revision-Date: %s\\n\"\n"+
			"\"Last-Translator: Vilmar Catafesta <vcatafesta@gmail.com>\\n\"\n"+
			"\"Language-Team: %s\\n\"\n"+
			"\"MIME-Version: 1.0\\n\"\n"+
			"\"Content-Type: text/plain; charset=UTF-8\\n\"\n"+
			"\"Content-Transfer-Encoding: 8bit\\n\"\n"+
			"\"Language: %s\\n\"\n"+
			"%s\n\n",
		_VERSION_, domain, domain, now, now, team, lang, pluralLine,
	)
	final := header + strings.Join(lines[bodyStart:], "\n")
	if err := os.WriteFile(path, []byte(final), 0644); err != nil {
		reportCmdError("stampPotHeader "+path, err)
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
			if !selfTestFlag && !quietFlag { // BUGFIX: --quiet não suprimia o [STATUS]
				fmt.Printf("\r    %s %s / %s %s", yellow(T("[STATUS]")), green(done), green(len(targetLangs)), T("idiomas concluídos..."))
			}
			muConsole.Unlock()
			<-sem
		}(lang)
	}
	wg.Wait()
}

// rateLimitMsg é o aviso exibido quando o motor bloqueia por excesso de pedidos.
func rateLimitMsg() string {
	return fmt.Sprintf(T("o motor '%s' bloqueou por excesso de pedidos (rate limiting). Tente mais tarde, use outro motor (-e bing) ou reduza -j."), engine)
}

// callUniversalTranslator traduz text para lang. Se não conseguir (offline ou falha
// do trans), devolve o texto original — comportamento usado por Markdown, man pages,
// JSON, HTML e texto puro, onde um trecho vazio quebraria o documento.
// Para .po use callTranslatorStrict, que informa a falha.
func callUniversalTranslator(text, lang string) string {
	// BUGFIX: antes aplicava TrimSpace aqui, e o fallback devolvia o texto já aparado.
	if res, ok := callTranslatorStrict(text, lang); ok {
		return res
	}
	return text
}

// splitOuterSpace separa os espaços iniciais e finais do miolo do texto.
func splitOuterSpace(s string) (lead, core, trail string) {
	rest := strings.TrimLeftFunc(s, unicode.IsSpace)
	lead = s[:len(s)-len(rest)]
	core = strings.TrimRightFunc(rest, unicode.IsSpace)
	trail = rest[len(core):]
	return lead, core, trail
}

// cacheNamespace devolve o "balde" do cache para o idioma. BUGFIX: a chave não
// considerava motor nem idioma de origem — trocar -e/-s reaproveitava traduções de
// outra combinação. Os padrões (google/auto) continuam usando só o idioma, para não
// invalidar o cache existente.
func cacheNamespace(lang string) string {
	if engine == "google" && sourceLang == "auto" {
		return lang
	}
	return lang + "|" + engine + "|" + sourceLang
}

// cacheKey monta a chave do cache. BUGFIX: antes era strings.ToLower(texto), então
// "Save" e "save" dividiam a mesma tradução (com a capitalização de quem entrou
// primeiro). Agora respeita maiúsculas e inclui as substituições de glossário
// aplicadas ao texto — se o glossário mudar, a tradução antiga não é reaproveitada.
func cacheKey(core string, placeholders map[string]string) string {
	var gl []string
	for i := 0; ; i++ {
		v, ok := placeholders[fmt.Sprintf("CHILI_GLOSS_%d_CHILI", i)]
		if !ok {
			break
		}
		gl = append(gl, v)
	}
	if len(gl) == 0 {
		return core
	}
	return core + "\x00glossary:" + strings.Join(gl, "\x1f")
}

// callTranslatorStrict traduz text para lang e informa se conseguiu.
// FEATURE: antes a falha era silenciosa (devolvia o original), e nos .po isso gerava
// msgstr idêntico ao msgid — o catálogo parecia traduzido, mas não estava.
// BUGFIX: espaços iniciais/finais (indentação, "\n" final de msgid, dois espaços de
// quebra de linha no Markdown) eram descartados; agora só o miolo vai ao motor e as
// bordas são recolocadas no resultado.
func callTranslatorStrict(text, lang string) (string, bool) {
	lead, core, trail := splitOuterSpace(text)
	if core == "" {
		if text == "" {
			return "", false
		}
		return text, true // só espaços: nada a traduzir
	}
	res, ok := translateCore(core, lang)
	if !ok {
		return "", false
	}
	return lead + res + trail, true
}

// translateCore traduz um texto já sem espaços nas bordas (cache, glossário, trans).
func translateCore(text, lang string) (string, bool) {
	// FEATURE (glossário): se o texto inteiro corresponde a um termo do glossário,
	// resolve direto sem cache nem rede — glossário sempre tem prioridade.
	if fixed, ok := glossaryExactMatch(text, lang); ok {
		atomic.AddInt64(&glossaryHits, 1)     // BUGFIX: antes não entrava nas estatísticas
		atomic.AddInt64(&fileGlossaryHits, 1) // BUGFIX: idem, por-arquivo
		return fixed, true
	}

	protectedText, placeholders := protectVariables(text, lang)
	key := cacheKey(text, placeholders)
	ns := cacheNamespace(lang)

	mu.Lock()
	if cacheData == nil {
		cacheData = make(map[string]map[string]CacheEntry)
	}
	if _, ok := cacheData[ns]; !ok {
		cacheData[ns] = make(map[string]CacheEntry)
	}
	if entry, exists := cacheData[ns][key]; exists && !forceFlag {
		entry.LastUsed = time.Now()
		cacheData[ns][key] = entry
		mu.Unlock()
		atomic.AddInt64(&cacheHits, 1)     // BUGFIX: contador atômico, fora do lock de cacheData
		atomic.AddInt64(&fileCacheHits, 1) // BUGFIX: estatística por-arquivo
		return entry.Value, true
	}
	mu.Unlock()

	// FEATURE (--dry-run): simula a tradução sem chamar rede nem gravar no cache.
	// Precisa vir ANTES do "if !isOnline", senão --dry-run fica mudo sem internet.
	// Aplica a mesma proteção (variáveis + glossário) e mostra o resultado.
	if dryRunFlag {
		atomic.AddInt64(&netCalls, 1)
		atomic.AddInt64(&fileNetCalls, 1)
		simulated := restoreVariables(protectedText, placeholders)
		return fmt.Sprintf("[DRY-RUN:%s] %s", lang, simulated), true
	}

	if !isOnline {
		atomic.AddInt32(&untranslated, 1)
		atomic.AddInt32(&fileUntranslated, 1)
		return "", false
	}

	// FEATURE: se o motor já bloqueou por excesso de pedidos nesta execução,
	// não insiste — cada nova chamada só pioraria o bloqueio.
	if atomic.LoadInt32(&rateLimited) == 1 {
		atomic.AddInt32(&untranslated, 1)
		atomic.AddInt32(&fileUntranslated, 1)
		return "", false
	}

	transLang := strings.ReplaceAll(lang, "_", "-")
	var res string
	for i := 0; i < 3; i++ {
		cmd := execCommand("trans", "-e", engine, "-s", sourceLang, "-no-init", "-no-autocorrect", "-b", ":"+transLang)
		cmd.Stdin = strings.NewReader(protectedText)
		// FEATURE: antes o stderr era descartado; o trans sai com código 0 mesmo
		// quando falha e explica o motivo só no stderr (ex.: rate limiting).
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, errCmd := cmd.Output()
		res = strings.TrimSpace(string(out))
		errMsg := strings.TrimSpace(stderr.String())

		if errCmd == nil && res != "" {
			res = restoreVariables(res, placeholders)
			break
		}
		res = ""
		if errMsg == "" && errCmd != nil {
			errMsg = errCmd.Error()
		}
		if errMsg != "" {
			lastTransErr.Store(strings.SplitN(errMsg, "\n", 2)[0])
		}
		// FEATURE: bloqueio por excesso de pedidos — para tudo e avisa uma vez.
		if strings.Contains(strings.ToLower(errMsg), "rate limit") {
			atomic.StoreInt32(&rateLimited, 1)
			rateLimitOnce.Do(func() {
				muConsole.Lock()
				fmt.Fprintf(os.Stderr, "\n%s %s\n", red(T("AVISO:")),
					white(rateLimitMsg()))
				muConsole.Unlock()
			})
			break
		}
		// FEATURE: espera crescente entre tentativas (1s, 2s), sem esperar após a última.
		if i < 2 {
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}
	if res == "" {
		atomic.AddInt32(&failedCalls, 1)
		atomic.AddInt32(&untranslated, 1)
		atomic.AddInt32(&fileUntranslated, 1)
		return "", false
	}
	atomic.AddInt64(&netCalls, 1)     // BUGFIX: antes era netCalls++ sem lock/atomic (data race)
	atomic.AddInt64(&fileNetCalls, 1) // BUGFIX: estatística por-arquivo
	mu.Lock()
	cacheData[ns][key] = CacheEntry{Value: res, LastUsed: time.Now()}
	mu.Unlock()
	return res, true
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
	markError() // BUGFIX (2.1.29): falhas de xgettext/msginit/msgfmt não afetavam o código de saída
	muConsole.Lock()
	fmt.Fprintf(os.Stderr, "%s %s: %v\n", red(T("[AVISO]")), white(step), err)
	muConsole.Unlock()
}

func prepareGettext(inputPath, baseName, lang string) {
	cleanName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	pot := filepath.Join("pot", cleanName+".pot")
	err := execCommand("xgettext", "--from-code=UTF-8", "--language="+lang, "--keyword=gettext", "--keyword=_", "--keyword=T", "--keyword=TN:1,2", "--force-po", "-o", pot, inputPath).Run()
	reportCmdError("xgettext ("+baseName+")", err)
	stampPotHeader(pot, cleanName, "")
}

func prepareGettextSelf(inputPath string) {
	pot := filepath.Join("pot", _APP_+".pot")
	err := execCommand("xgettext", "--from-code=UTF-8", "--keyword=T", "--keyword=TN:1,2", "--no-wrap", "-o", pot, inputPath).Run()
	reportCmdError("xgettext --self", err)
	stampPotHeader(pot, _APP_, "")
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

// reRoffEscape casa escapes roff que não podem ir ao motor de tradução: mudanças de
// fonte (\fB, \f(CW, \f[CR]), caracteres especiais (\(bu, \[em]), \- \e \& \| \^ \~
// \c, espaçamentos (\h'..', \s-1) e referências a strings/registradores (\*x, \n(xx).
var reRoffEscape = regexp.MustCompile(`\\f(?:\[[^\]]*\]|\([A-Za-z0-9]{2}|[A-Za-z0-9])|\\\([^\s]{2}|\\\[[^\]]*\]|\\[*n](?:\[[^\]]*\]|\([A-Za-z0-9]{2}|[A-Za-z0-9])|\\[hvwsoHSDlLN]\x27[^\x27]*\x27|\\s[-+]?[0-9]+|\\[-e&|^~c0 ]`)

// reRoffOnly casa o que sobra de uma linha que não tem texto traduzível.
var reRoffOnly = regexp.MustCompile(`CHILI_ROFF_[0-9]+_CHILI|[\s\p{P}]`)

// manTextMacros são as macros cujos argumentos são texto corrido traduzível.
// As demais (.TH, .TP, .IP, .BR, .UR, .so, .de, .ft...) ficam intactas.
var manTextMacros = map[string]bool{".SH": true, ".SS": true, ".B": true, ".I": true, ".SM": true, ".SB": true}

// translateRoffText traduz uma linha de texto roff protegendo os escapes.
func translateRoffText(text, lang string) string {
	marks := make(map[string]string)
	n := 0
	protected := reRoffEscape.ReplaceAllStringFunc(text, func(esc string) string {
		p := fmt.Sprintf("CHILI_ROFF_%d_CHILI", n)
		n++
		marks[p] = esc
		return p
	})
	// só placeholders/pontuação: nada a traduzir
	if strings.TrimSpace(reRoffOnly.ReplaceAllString(protected, "")) == "" {
		return text
	}
	return restoreVariables(callUniversalTranslator(protected, lang), marks)
}

// translateManPage traduz uma página de manual roff.
// BUGFIX (2.1.29): antes traduzia os argumentos de TODAS as macros (.TH, .BR ls (1),
// .IP \(bu 4...), traduzia comentários .\" (o teste olhava o resto da linha, não a
// macro), mandava escapes roff sem proteção ao motor e juntava espaços com
// strings.Fields. Agora só traduz texto corrido e os argumentos de .SH/.SS/.B/.I/.SM/.SB,
// preserva comentários e blocos sem preenchimento (.nf/.fi, .EX/.EE) e o espaçamento.
func translateManPage(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil { // BUGFIX: erro de leitura era ignorado e gerava saída vazia
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível ler")), yellow(inputPath), err)
		markError()
		return
	}
	lines := strings.Split(string(content), "\n")
	var translatedLines []string
	noFill := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			translatedLines = append(translatedLines, line)
		// strings normais (não crase) para não confundir o parser C do xgettext em --self
		case strings.HasPrefix(trimmed, ".\\\"") || strings.HasPrefix(trimmed, "'\\\"") ||
			strings.HasPrefix(trimmed, "\\\"") || strings.HasPrefix(trimmed, ".\\#") || strings.HasPrefix(trimmed, "\\#"):
			translatedLines = append(translatedLines, line) // comentário
		case strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "'"):
			macro, rest := trimmed, ""
			if idx := strings.IndexAny(trimmed, " \t"); idx > 0 {
				macro, rest = trimmed[:idx], strings.TrimLeft(trimmed[idx:], " \t")
			}
			switch macro {
			case ".nf", ".EX":
				noFill = true
			case ".fi", ".EE":
				noFill = false
			}
			if !manTextMacros[macro] || rest == "" || noFill {
				translatedLines = append(translatedLines, line)
				continue
			}
			// argumento entre aspas: traduz o conteúdo e recoloca as aspas
			if len(rest) >= 2 && strings.HasPrefix(rest, "\"") && strings.HasSuffix(rest, "\"") {
				translatedLines = append(translatedLines, macro+" \""+translateRoffText(rest[1:len(rest)-1], lang)+"\"")
			} else {
				translatedLines = append(translatedLines, macro+" "+translateRoffText(rest, lang))
			}
		case noFill:
			translatedLines = append(translatedLines, line) // exemplo/código: intacto
		default:
			translatedLines = append(translatedLines, translateRoffText(line, lang))
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
		markError()
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translateHTML(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil { // BUGFIX: erro de leitura era ignorado e gerava saída vazia
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível ler")), yellow(inputPath), err)
		markError()
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
		markError()
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translateMarkdown(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil { // BUGFIX: erro de leitura era ignorado e gerava saída vazia
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível ler")), yellow(inputPath), err)
		markError()
		return
	}
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
		markError()
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

func translatePlaintext(inputPath, lang string) {
	content, err := os.ReadFile(inputPath)
	if err != nil { // BUGFIX: erro de leitura era ignorado e gerava saída vazia
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível ler")), yellow(inputPath), err)
		markError()
		return
	}
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
		markError()
	}
	updateProgress(lang, len(lines), len(lines), "OK")
}

// poUnescape converte uma linha de string PO (ex: "Ola \"mundo\"\n") no texto real.
func poUnescape(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "\"")
	raw = strings.TrimSuffix(raw, "\"")
	// BUGFIX (2.1.29): antes usava ReplaceAll em sequência (primeiro barra+n, depois
	// barra dupla), então uma barra invertida literal seguida de "n" no msgid (comum
	// em scripts bash que passam "texto\\n" ao printf) virava barra + quebra de linha
	// real, e o msgfmt rejeitava o .po. Agora decodifica em uma passada só.
	var sb strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' || i+1 >= len(raw) {
			sb.WriteByte(c)
			continue
		}
		i++
		switch raw[i] {
		case 'n':
			sb.WriteByte('\n')
		case 't':
			sb.WriteByte('\t')
		case 'r':
			sb.WriteByte('\r')
		case 'a':
			sb.WriteByte('\a')
		case 'b':
			sb.WriteByte('\b')
		case 'f':
			sb.WriteByte('\f')
		case 'v':
			sb.WriteByte('\v')
		case '\\', '"':
			sb.WriteByte(raw[i])
		default:
			sb.WriteByte('\\')
			sb.WriteByte(raw[i])
		}
	}
	return sb.String()
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
	text = strings.ReplaceAll(text, "\r", "\\r") // BUGFIX (2.1.29): \r saía cru no .po
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

	stampPotHeader(poTmp, cleanBase, lang)

	file, err := os.Open(poTmp)
	if err != nil {
		// BUGFIX (BUG-09): erro de abertura não pode ser silenciado — sem isso o idioma
		// falha silenciosamente sem gerar .po nem aviso nenhum.
		logVerbose("translateFile: falha ao abrir %s: %v", poTmp, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível abrir")), yellow(poTmp), err)
		markError()
		return
	}
	defer file.Close()

	output, errCreate := os.Create(poFinal)
	if errCreate != nil {
		logVerbose("translateFile: falha ao criar %s: %v", poFinal, errCreate)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível criar")), yellow(poFinal), errCreate)
		markError()
		return
	}
	defer output.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024) // BUGFIX: linhas > 64 KiB abortavam a leitura calada
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		reportCmdError("leitura de "+poTmp, err)
		markError()
		return
	}

	// BUGFIX (2.1.29): o corpo era identificado pela primeira linha "#:"; .po sem
	// comentários de referência não tinha nada traduzido. Agora o fim do cabeçalho
	// (entrada msgid "") é detectado por poHeaderEnd.
	bodyStart, _, _ := poHeaderEnd(lines)

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
	for _, l := range lines[bodyStart:] {
		if (strings.HasPrefix(l, "msgstr ") || strings.HasPrefix(l, "msgstr[")) {
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
	skipMsgstrCont := false // BUGFIX (2.1.30): descarta a continuação do msgstr antigo

	for lineIdx, line := range lines {
		if lineIdx == bodyStart {
			pastHeader = true
		}

		// BUGFIX (2.1.30): o msgstr original pode ter linhas de continuação (ex: para
		// -l en o msginit já preenche msgstr com cópia do msgid, e msgids longos saem
		// em várias linhas). O msgstr é regravado inteiro em uma linha, então essas
		// continuações antigas caíam no default e eram coladas depois da tradução nova,
		// gerando msgstr = tradução + original (msgfmt -c acusava erro de formato).
		if skipMsgstrCont {
			if strings.HasPrefix(strings.TrimSpace(line), "\"") {
				continue
			}
			skipMsgstrCont = false
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
			// FEATURE: sem tradução, msgstr fica vazio (padrão gettext para "pendente"),
			// em vez de copiar o msgid e mascarar a falha.
			translated, ok := callTranslatorStrict(original, lang)
			if !ok {
				translated = ""
			}
			fmt.Fprintf(output, "msgstr[%d] \"%s\"\n", idx, poEscape(translated))
			skipMsgstrCont = true

		case pastHeader && strings.HasPrefix(line, "msgstr ") && haveEntry:
			current++
			updateProgress(lang, current, totalMsgids, "PO")
			original := poJoinMsgid(msgidLines)
			// FEATURE: idem — sem tradução, msgstr vazio em vez de cópia do msgid.
			translated, ok := callTranslatorStrict(original, lang)
			if !ok {
				translated = ""
			}
			fmt.Fprintf(output, "msgid %s\nmsgstr \"%s\"\n", strings.Join(msgidLines, "\n"), poEscape(translated))
			skipMsgstrCont = true
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
		markError()
		return
	}

	var out []byte
	if isYAML {
		// BUGFIX (2.1.29): antes decodificava para interface{} (map), o que perdia a
		// ordem das chaves e todos os comentários. yaml.Node preserva os dois.
		var doc yaml.Node
		if err := yaml.Unmarshal(data, &doc); err != nil {
			logVerbose("translateJSON: YAML inválido em %s: %v", path, err)
			fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("YAML inválido em")), yellow(path), err)
			markError()
			return
		}
		translateYAMLNode(&doc, lang, false)
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		err = enc.Encode(&doc)
		enc.Close()
		out = buf.Bytes()
	} else {
		// BUGFIX (2.1.29): antes decodificava para interface{}: a ordem das chaves se
		// perdia (Go ordena alfabeticamente), números viravam float64 (inteiros acima
		// de 2^53 perdiam precisão) e <, > e & saíam como < etc.
		var v interface{}
		v, err = decodeOrderedJSON(data)
		if err != nil {
			logVerbose("translateJSON: JSON inválido em %s: %v", path, err)
			fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("JSON inválido em")), yellow(path), err)
			markError()
			return
		}
		v = translateValue(v, lang)
		var buf bytes.Buffer
		err = encodeOrderedJSON(&buf, v, "")
		buf.WriteString("\n")
		out = buf.Bytes()
	}
	if err != nil {
		logVerbose("translateJSON: erro serializando saída de %s: %v", path, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Falha ao gerar saída para")), yellow(path), err)
		markError()
		return
	}

	outFile := filepath.Join(targetDir, fmt.Sprintf("%s-%s%s", strings.TrimSuffix(filepath.Base(path), ext), lang, ext))
	if err := dryWriteFile(outFile, out); err != nil { // FEATURE (--dry-run)
		logVerbose("translateJSON: erro gravando %s: %v", outFile, err)
		fmt.Printf("%s %s '%s' (%s)\n", red(T("ERRO:")), white(T("Não foi possível gravar")), yellow(outFile), err)
		markError()
		return
	}
	updateProgress(lang, 100, 100, "OK") // BUGFIX: os outros formatos terminam em "OK"
}

// orderedObject é um objeto JSON que mantém a ordem original das chaves.
type orderedObject struct {
	keys []string
	vals map[string]interface{}
}

// decodeOrderedJSON decodifica JSON mantendo ordem das chaves e números como json.Number.
func decodeOrderedJSON(data []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeJSONValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("conteúdo extra após o JSON")
	}
	return v, nil
}

func decodeJSONValue(dec *json.Decoder) (interface{}, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := &orderedObject{vals: make(map[string]interface{})}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("chave inválida: %v", kt)
				}
				val, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				if _, dup := obj.vals[key]; !dup {
					obj.keys = append(obj.keys, key)
				}
				obj.vals[key] = val
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return obj, nil
		case '[':
			arr := []interface{}{}
			for dec.More() {
				val, err := decodeJSONValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return arr, nil
		}
		return nil, fmt.Errorf("delimitador inesperado: %v", t)
	default:
		return tok, nil // string, json.Number, bool, nil
	}
}

// jsonString codifica uma string JSON sem escapar <, > e &.
func jsonString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimRight(buf.String(), "\n")
}

// encodeOrderedJSON grava v com indentação de 2 espaços, na ordem original.
func encodeOrderedJSON(w *bytes.Buffer, v interface{}, indent string) error {
	inner := indent + "  "
	switch t := v.(type) {
	case *orderedObject:
		if len(t.keys) == 0 {
			w.WriteString("{}")
			return nil
		}
		w.WriteString("{\n")
		for i, k := range t.keys {
			w.WriteString(inner + jsonString(k) + ": ")
			if err := encodeOrderedJSON(w, t.vals[k], inner); err != nil {
				return err
			}
			if i < len(t.keys)-1 {
				w.WriteString(",")
			}
			w.WriteString("\n")
		}
		w.WriteString(indent + "}")
	case []interface{}:
		if len(t) == 0 {
			w.WriteString("[]")
			return nil
		}
		w.WriteString("[\n")
		for i, e := range t {
			w.WriteString(inner)
			if err := encodeOrderedJSON(w, e, inner); err != nil {
				return err
			}
			if i < len(t)-1 {
				w.WriteString(",")
			}
			w.WriteString("\n")
		}
		w.WriteString(indent + "]")
	case string:
		w.WriteString(jsonString(t))
	case json.Number:
		w.WriteString(t.String()) // número exatamente como no original
	case bool:
		w.WriteString(strconv.FormatBool(t))
	case nil:
		w.WriteString("null")
	default:
		return fmt.Errorf("tipo inesperado %T", v)
	}
	return nil
}

// reNotProse casa valores que não são texto para pessoas e não devem ser traduzidos.
var reNotProse = regexp.MustCompile(`^(?:` +
	`[a-zA-Z][a-zA-Z0-9+.-]*://\S*` + // URL
	`|[^\s@]+@[^\s@]+\.[a-zA-Z]{2,}` + // e-mail
	`|\S*/\S*` + // caminho (tem "/" e nenhum espaço)
	`|[\w.-]+\.[a-zA-Z0-9]{1,5}` + // nome de arquivo (icon.png, main.go)
	`|[a-z0-9]+(?:[_.-][a-z0-9]+)+` + // identificador snake/kebab/dotted (save_button, app.title)
	`|[A-Z0-9]+(?:_[A-Z0-9]+)+` + // CONSTANTE
	`|#?[0-9a-fA-F]{6,}` + // hash, cor hex
	`|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}` + // UUID
	`)$`)

// isTranslatableValue decide se um valor de JSON/YAML parece texto para pessoas.
// BUGFIX (2.1.29): antes TODAS as strings eram traduzidas, inclusive IDs, caminhos,
// URLs e nomes de ícones/arquivos.
func isTranslatableValue(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" || !strings.ContainsFunc(t, unicode.IsLetter) {
		return false // vazio, números, datas, pontuação
	}
	return !reNotProse.MatchString(t)
}

// translateValue percorre recursivamente objetos, arrays e strings do JSON.
// BUGFIX (BUG-05): strings dentro de arrays também são traduzidas.
func translateValue(v interface{}, lang string) interface{} {
	switch val := v.(type) {
	case string:
		if !isTranslatableValue(val) {
			return val
		}
		return callUniversalTranslator(val, lang)
	case *orderedObject:
		for _, k := range val.keys {
			val.vals[k] = translateValue(val.vals[k], lang)
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

// translateYAMLNode traduz os valores string de um documento YAML (nunca as chaves).
func translateYAMLNode(n *yaml.Node, lang string, isKey bool) {
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			translateYAMLNode(c, lang, false)
		}
	case yaml.MappingNode:
		for i, c := range n.Content {
			translateYAMLNode(c, lang, i%2 == 0) // posições pares são chaves
		}
	case yaml.ScalarNode:
		if isKey || n.Tag != "!!str" || !isTranslatableValue(n.Value) {
			return
		}
		n.Value = callUniversalTranslator(n.Value, lang)
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
			// BUGFIX: antes qualquer ":" no valor ativava o formato por idioma, então
			// "site=https://chililinux.com" virava {"https": "//chililinux.com"} e
			// "Nota=Obs: importante" virava {"obs": "importante"}, perdendo a tradução
			// global. Agora só é por idioma quando TODAS as chaves são códigos de idioma.
			if pl, ok := parsePerLang(val); ok {
				perLang = pl
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
// normLangCode normaliza um código de idioma para comparação: minúsculas e "_".
func normLangCode(c string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c), "-", "_"))
}

// isKnownLangCode diz se c é um idioma suportado (ex: "pt_br", "zh-TW") ou o
// prefixo de um (ex: "pt", "zh").
func isKnownLangCode(c string) bool {
	c = normLangCode(c)
	if c == "" {
		return false
	}
	for _, l := range supportedLanguages {
		n := normLangCode(l)
		if c == n || strings.HasPrefix(n, c+"_") {
			return true
		}
	}
	return false
}

// parsePerLang interpreta "en:Product;fr:Produit". Só aceita quando todos os pares
// têm um código de idioma conhecido antes do ":"; caso contrário o valor é global.
func parsePerLang(val string) (map[string]string, bool) {
	if !strings.Contains(val, ":") {
		return nil, false
	}
	perLang := make(map[string]string)
	for _, pair := range strings.Split(val, ";") {
		if strings.TrimSpace(pair) == "" {
			continue
		}
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 || !isKnownLangCode(kv[0]) {
			return nil, false
		}
		perLang[normLangCode(kv[0])] = strings.TrimSpace(kv[1]) // BUGFIX: "zh-TW" casa com zh_TW
	}
	if len(perLang) == 0 {
		return nil, false
	}
	return perLang, true
}

func resolveGlossaryTranslation(entry glossaryEntry, lang string) string {
	if entry.perLang != nil {
		normLang := normLangCode(lang)
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

// reProtect casa trechos que nunca devem ir ao motor de tradução: variáveis de shell,
// especificadores printf, código inline, links/imagens Markdown e URLs.
// BUGFIX: o padrão de % aceitava espaço como flag e qualquer letra como conversão,
// então "100% de" virava "100CHILI_REF_0_CHILIe". Agora exige uma conversão printf
// válida (com modificador de tamanho opcional), aceita posicional (%1$s) e %%.
var reProtect = regexp.MustCompile(
	`\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%%|` +
		`%(?:[0-9]+\$)?[-+#0]*(?:[0-9]+|\*)?(?:\.(?:[0-9]+|\*))?(?:hh|h|ll|l|L|q|j|z|t)?[diouxXeEfFgGaAcsSpnbqvwT]|` +
		"`[^`\\n]+`" + `|!\[.*?\]\(.*?\)|\[.*?\]\(.*?\)|https?://[^\s]+`)

// protectVariables troca os trechos protegidos por placeholders CHILI_REF_N_CHILI e,
// em seguida, os termos do glossário por CHILI_GLOSS_N_CHILI.
// BUGFIX: antes o glossário rodava primeiro; um termo dentro de crases, link ou URL
// virava placeholder dentro de outro placeholder, e como a restauração percorria um
// map (ordem aleatória) o resultado às vezes ficava com CHILI_GLOSS_N_CHILI no meio.
// Agora código/URLs são protegidos primeiro (e ficam literais, sem glossário), em uma
// única passada por posição (antes strings.Replace podia trocar a ocorrência errada).
func protectVariables(text, lang string) (string, map[string]string) {
	placeholders := make(map[string]string)
	var sb strings.Builder
	last, n := 0, 0
	for _, m := range reProtect.FindAllStringIndex(text, -1) {
		sb.WriteString(text[last:m[0]])
		p := fmt.Sprintf("CHILI_REF_%d_CHILI", n)
		n++
		placeholders[p] = text[m[0]:m[1]]
		sb.WriteString(p)
		last = m[1]
	}
	sb.WriteString(text[last:])
	protected := protectGlossaryTerms(sb.String(), lang, placeholders)
	return protected, placeholders
}

// restoreVariables devolve os trechos originais no lugar dos placeholders. Repete
// enquanto houver substituição, então a ordem de iteração do map não importa.
func restoreVariables(text string, p map[string]string) string {
	for pass := 0; pass < 3; pass++ {
		changed := false
		for k, v := range p {
			if strings.Contains(text, k) {
				text = strings.ReplaceAll(text, k, v)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
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

// checkInternet considera online se QUALQUER destino responder.
// FEATURE: antes testava só 8.8.8.8:53, que muitos provedores/firewalls bloqueiam,
// e o programa se achava offline mesmo com internet funcionando.
func checkInternet() bool {
	targets := []string{
		"8.8.8.8:53",
		"1.1.1.1:53",
		"translate.googleapis.com:443",
	}
	for _, addr := range targets {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// detectDistro devolve a família da distribuição para escolher o gerenciador de
// pacotes. BUGFIX (2.1.29): só olhava ID=, então derivadas (VoidBR, Manjaro, Mint...)
// não recebiam a oferta de instalação; agora também consulta ID_LIKE.
func detectDistro() string {
	osData, _ := os.ReadFile("/etc/os-release")
	var ids []string
	for _, line := range strings.Split(string(osData), "\n") {
		for _, key := range []string{"ID=", "ID_LIKE="} {
			if strings.HasPrefix(line, key) {
				val := strings.Trim(strings.TrimPrefix(line, key), "\"' ")
				ids = append(ids, strings.Fields(strings.ToLower(val))...)
			}
		}
	}
	for _, id := range ids {
		switch id {
		case "chili", "chililinux", "arch", "void", "voidbr", "debian", "ubuntu", "fedora":
			return id
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
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
	case "void", "voidbr": installCmd = "sudo xbps-install -S " + pkgList
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
	if err != nil {
		return
	}
	if err := json.Unmarshal(file, &cacheData); err != nil {
		// BUGFIX (2.1.29): antes o erro era ignorado e o cache corrompido era
		// sobrescrito no fim da execução, perdendo todas as traduções sem aviso.
		cacheData = make(map[string]map[string]CacheEntry)
		bad := cacheFile + ".bad"
		os.Rename(cacheFile, bad)
		fmt.Fprintf(os.Stderr, "%s %s '%s' (%v)\n", yellow(T("[AVISO]")), white(T("Cache corrompido; cópia guardada em")), yellow(bad), err)
	}
}

// saveCache grava o cache de forma atômica. BUGFIX (2.1.29): antes gravava direto
// sobre o cache.json; uma interrupção no meio deixava o arquivo inválido.
func saveCache() {
	mu.Lock()
	defer mu.Unlock()
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		reportCmdError("saveCache", err)
		return
	}
	tmp := cacheFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		reportCmdError("saveCache", err)
		return
	}
	if err := os.Rename(tmp, cacheFile); err != nil {
		reportCmdError("saveCache", err)
	}
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

// potPathFor devolve o caminho do .pot gerado/copiado para o arquivo de entrada.
// BUGFIX: hasActualContent e cleanupEmpty usavam pot/<nome>.<ext>.pot, mas o xgettext
// grava pot/<nome>.pot (e o .pot de entrada é copiado como pot/<nome>.pot).
func potPathFor(ext, baseName string) string {
	if selfFlag {
		return filepath.Join("pot", _APP_+".pot")
	}
	return filepath.Join("pot", strings.TrimSuffix(baseName, ext)+".pot")
}

func hasActualContent(ext, baseName string) bool {
	if selfFlag || dryRunFlag {
		return true
	}
	isMan, _ := regexp.MatchString(`^\.[1-9]$`, ext)
	if isMan { return true }
	if ext == ".md" || ext == ".markdown" || ext == ".txt" || ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".html" || ext == ".htm" { return true }
	content, err := os.ReadFile(potPathFor(ext, baseName))
	if err != nil {
		return true
	}
	// BUGFIX: antes bastava conter "msgid" — o msgid "" do cabeçalho sempre casava.
	lines := strings.Split(string(content), "\n")
	bodyStart, _, _ := poHeaderEnd(lines)
	for _, l := range lines[bodyStart:] {
		if strings.HasPrefix(l, "msgid ") {
			return true
		}
	}
	return false
}

func cleanupEmpty(ext, baseName string) {
	if dryRunFlag {
		return
	}
	os.Remove(potPathFor(ext, baseName))
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
	// BUGFIX (2.1.29): o "em" estava fixo no código, fora do T().
	fmt.Printf("\n\n%s %s | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n",
		green("✔"), white(fmt.Sprintf(T("Concluído em %v"), time.Since(start).Round(time.Second))),
		blue(T("Cache:")), hits, pCache,
		yellow(T("Net:")), netVal, pNet,
		magenta(T("Glossário:")), gloss, pGloss,
		white(T("Total:")), total)
	// FEATURE: deixa visível quando algo ficou sem tradução (antes era silencioso).
	if n := atomic.LoadInt32(&fileUntranslated); n > 0 {
		fmt.Printf("%s %s: %s %s\n", red("✘"), white(T("Não traduzidos")), red(n),
			T("(ficaram no idioma original; nos .po, com msgstr vazio — rode de novo quando a conexão/trans estiver ok)"))
		if e, ok := lastTransErr.Load().(string); ok && e != "" {
			fmt.Printf("%s %s: %s\n", red("✘"), white(T("Último erro do trans")), yellow(e))
		}
		// FEATURE: o aviso impresso durante o processamento é sobrescrito pelo
		// redesenho das barras de progresso; repete aqui, onde não se perde.
		if atomic.LoadInt32(&rateLimited) == 1 {
			fmt.Printf("%s %s %s\n", red("✘"), red(T("AVISO:")), white(rateLimitMsg()))
		}
	}
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
	if n := atomic.LoadInt32(&untranslated); n > 0 {
		fmt.Printf("    → %-15s: %s\n", T("Não traduzidos"), red(n))
		if e, ok := lastTransErr.Load().(string); ok && e != "" {
			fmt.Printf("    → %-15s: %s\n", T("Último erro"), yellow(e))
		}
		if atomic.LoadInt32(&rateLimited) == 1 {
			fmt.Printf("    → %-15s: %s\n", T("AVISO"), red(rateLimitMsg()))
		}
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
	}	// FEATURE (2.1.29): documenta o código de saída
	fmt.Fprintf(os.Stderr, "\n%s: %s\n", yellow(T("Código de saída")),
		white(T("0 = ok, 1 = erro (arquivo ausente, leitura/gravação), 2 = ficaram textos sem tradução")))
}
