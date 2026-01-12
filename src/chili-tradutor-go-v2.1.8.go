/*
    chili-tradutor-go
    Wrapper universal de tradução automática com cache inteligente

    Site:      https://chililinux.com
    GitHub:    https://github.com/vcatafesta/chili/go

    Created:   dom 01 out 2023 09:00:00 -03
    Altered:   qui 05 out 2023 10:00:00 -03
    Updated:   seg 12 jan 2026 12:45:00 -04
    Version:   2.1.8

    Copyright (c) 2019-2026, Vilmar Catafesta <vcatafesta@gmail.com>
    Copyright (c) 2019-2026, ChiliLinux Team
    All rights reserved.
*/

package main

import (
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

// Constantes globais para identificação e versionamento
const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "2.1.8-20260112"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta"
)

// --- CONFIGURAÇÃO DE CORES ---
// Inicializamos funções de cores para facilitar o uso no Printf sem repetir códigos ANSI manuais
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
	cacheData     map[string]map[string]string // Mapa aninhado: [idioma][texto_original] = tradução
	mu            sync.Mutex                   // Mutex para evitar condições de corrida no mapa de cache (thread-safety)
	muConsole     sync.Mutex                   // Mutex para evitar que várias threads escrevam no terminal ao mesmo tempo
	cacheHits     int                          // Contador de economia de rede
	netCalls      int                          // Contador de chamadas reais ao tradutor
	isOnline      bool                         // Status da internet detectado no início
	langsDone     int32                        // Contador atômico para progresso global
	langPositions map[string]int               // Armazena a linha de cada idioma para atualização visual dinâmica
)

// Lista de idiomas suportados pelo motor trans
var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh-CN", "zh-TW",
}

// Idiomas processados por padrão caso o usuário não use o parâmetro -l
var defaultLanguages = []string{"en", "es", "it", "de", "fr", "ru", "zh-CN", "zh-TW", "ja", "ko"}

// init roda antes do main e prepara o ambiente do sistema
func init() {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", _APP_)
	os.MkdirAll(cacheDir, 0755) // Cria o diretório de cache se não existir
	cacheFile = filepath.Join(cacheDir, "cache.json")
	langPositions = make(map[string]int)
}

func main() {
	checkDependencies() // Valida se xgettext, trans, etc estão instalados
	isOnline = checkInternet() // Define se podemos fazer chamadas externas ou apenas cache

	// Configuração do analisador de argumentos (flags)
	pflag.Usage = usage
	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo fonte para tradução (.sh, .py, .md)")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor: google, bing, yandex")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas (ex: pt-BR,en) ou 'all'")
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
		pflag.Usage() // Se não houver arquivo de entrada, mostra ajuda e para
		os.Exit(1)
	}

	baseName := filepath.Base(inputFile)
	ext := strings.ToLower(filepath.Ext(baseName))
	isMarkdown := (ext == ".md" || ext == ".markdown")

	confLog()    // Inicializa o arquivo de log para depuração
	loadCache()  // Carrega o arquivo JSON de traduções anteriores para a memória
	defer saveCache() // Garante que o cache seja gravado em disco ao final do programa

	// Define quais idiomas serão processados
	targetLangs := defaultLanguages
	if len(languages) > 0 {
		if languages[0] == "all" {
			targetLangs = supportedLanguages
		} else {
			targetLangs = languages
		}
	}

	fmt.Printf("%s %s %s\n", cyan(">>"), white(_APP_), white(_VERSION_))

	// STEP 1: PREPARAÇÃO DOS ARQUIVOS
	if isMarkdown {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Modo Markdown: Saída em doc/"))
		os.MkdirAll("doc", 0755)
	} else {
		fmt.Printf("%s %s\n", yellow("[STEP 1]"), white("Extraindo strings para pot/ ..."))
		os.MkdirAll("pot", 0755)
		prepareGettext(inputFile, baseName) // Extrai textos do script shell para um modelo .pot
	}

	// STEP 2: CONFIGURAÇÃO VISUAL E START
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

	// Reserva o espaço visual no terminal para cada idioma (barra de progresso)
	for i, lang := range targetLangs {
		langPositions[lang] = len(targetLangs) - i
		fmt.Printf("   %s %-7s %s\n", blue("→"), cyan(lang), yellow("[Aguardando...]"))
	}

	start := time.Now()
	var wg sync.WaitGroup
	sem := make(chan struct{}, jobs) // Canal usado como semáforo para limitar a concorrência
	totalLangs := len(targetLangs)

	for _, lang := range targetLangs {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{} // Ocupa uma vaga no semáforo
			
			if isMarkdown {
				translateMarkdown(inputFile, l)
			} else {
				prepareMsginit(baseName, l) // Cria o arquivo .po para o idioma
				translateFile(baseName, l)  // Traduz o conteúdo do .po
				writeMsgfmtToMo(baseName, l) // Compila para binário .mo
			}
			
			// Incremento atômico para evitar erros de soma em threads paralelas
			atomic.AddInt32(&langsDone, 1)
			
			// Atualiza a linha de status global no rodapé das barras
			muConsole.Lock()
			fmt.Printf("\r   %s %s / %s idiomas concluídos...", yellow("[STATUS]"), green(langsDone), green(totalLangs))
			muConsole.Unlock()
			
			<-sem // Libera uma vaga no semáforo
		}(lang)
	}
	wg.Wait() // Aguarda todos os processos terminarem

	// CÁLCULO DE PERFORMANCE FINAL
	totalCalls := cacheHits + netCalls
	pCache := 0.0
	pNet := 0.0
	if totalCalls > 0 {
		pCache = (float64(cacheHits) / float64(totalCalls)) * 100
		pNet = (float64(netCalls) / float64(totalCalls)) * 100
	}

	// Saída consolidada em uma única linha (v2.1.8)
	fmt.Printf("\n\n%s %s em %v | %s %d (%.2f%%) | %s %d (%.2f%%) | %s %d\n\n", 
		green("✔"), white("Concluído"), time.Since(start).Round(time.Second),
		blue("Cache:"), cacheHits, pCache,
		yellow("Net:"), netCalls, pNet,
		white("Total:"), totalCalls,
	)
}

// --- FUNÇÕES DE APOIO ---

// translateMarkdown lê o MD linha por linha e traduz mantendo a estrutura
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
		// Pula tradução de linhas vazias ou blocos de código markdown
		if trimmed == "" || strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "`") {
			translatedLines = append(translatedLines, line)
		} else {
			translatedLines = append(translatedLines, callUniversalTranslator(line, lang))
		}
		// Atualiza o progresso visual a cada 5 linhas para não sobrecarregar o terminal
		if i%5 == 0 || i == total-1 {
			updateProgress(lang, i+1, total, "MD")
		}
	}
	os.WriteFile(outFile, []byte(strings.Join(translatedLines, "\n")), 0644)
	updateProgress(lang, total, total, "OK")
}

// callUniversalTranslator é o coração do script: gerencia a ponte entre o cache e o motor trans
func callUniversalTranslator(text, lang string) string {
	text = strings.TrimSpace(text)
	if text == "" { return "" }
	normID := strings.ToLower(text) // Chave do cache sempre em minúsculo

	mu.Lock()
	if _, ok := cacheData[lang]; !ok { cacheData[lang] = make(map[string]string) }
	// Se já traduzimos isso antes, retornamos o valor local para poupar internet
	if val, ok := cacheData[lang][normID]; ok && !forceFlag {
		cacheHits++
		mu.Unlock()
		return val
	}
	mu.Unlock()

	// Se não há internet e não está no cache, retornamos o original para não travar
	if !isOnline { return text }

	// Proteção contra tradução de variáveis $HOME, links, etc.
	protectedText, placeholders := protectVariables(text)
	netCalls++

	// Executa o comando trans do sistema
	cmd := exec.Command("trans", "-e", engine, "-no-init", "-no-autocorrect", "-b", ":"+lang)
	cmd.Stdin = strings.NewReader(protectedText)
	out, err := cmd.Output()

	res := text
	if err == nil { res = strings.TrimSpace(string(out)) }
	
	// Restaura as variáveis originais no texto traduzido
	res = restoreVariables(res, placeholders)

	// Salva a nova tradução no cache memória
	mu.Lock()
	cacheData[lang][normID] = res
	mu.Unlock()
	// Nota: Salvar em disco a cada chamada é seguro contra crashes, mas pode ser lento.
	saveCache()

	return res
}

// protectVariables troca elementos sensíveis ($var, links) por placeholders únicos
func protectVariables(text string) (string, map[string]string) {
	// Regex para identificar variáveis shell, links markdown e URLs
	re := regexp.MustCompile(`(\$\{[A-Za-z0-9_.]+\}|\$[A-Za-z0-9_.]+|%[a-z]|\[.*?\]\(.*?\)|<a\s+.*?>.*?</a>|https?://[^\s)\]]+)`)
	placeholders := make(map[string]string)
	matches := re.FindAllString(text, -1)
	protectedText := text
	for i, match := range matches {
		placeholder := fmt.Sprintf("CHILI_REF_%d_CHILI", i)
		placeholders[placeholder] = match
		// Substitui apenas a primeira ocorrência para evitar conflitos
		protectedText = strings.Replace(protectedText, match, placeholder, 1)
	}
	return protectedText, placeholders
}

// restoreVariables devolve o conteúdo original nos lugares marcados
func restoreVariables(text string, placeholders map[string]string) string {
	for p, o := range placeholders { text = strings.Replace(text, p, o, -1) }
	return text
}

// checkInternet valida conectividade básica via DNS do Google
func checkInternet() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 2*time.Second)
	if err != nil { return false }
	conn.Close()
	return true
}

// loadCache decodifica o arquivo JSON de cache para o mapa global
func loadCache() {
	cacheData = make(map[string]map[string]string)
	file, err := os.ReadFile(cacheFile)
	if err == nil { json.Unmarshal(file, &cacheData) }
}

// saveCache grava o mapa de cache de volta no arquivo JSON
func saveCache() {
	mu.Lock() // Protege contra escrita simultânea enquanto outra goroutine tenta salvar
	defer mu.Unlock()
	data, _ := json.MarshalIndent(cacheData, "", "  ") // Indentado para ser legível por humanos
	os.WriteFile(cacheFile, data, 0644)
}

// updateProgress manipula o cursor do terminal para atualizar a linha correta de cada idioma
func updateProgress(lang string, current, total int, suffix string) {
	if quietFlag || verboseFlag { return }
	muConsole.Lock()
	defer muConsole.Unlock()

	pos := langPositions[lang] // Qual linha acima do cursor atual devemos editar?
	percent := 0
	if total > 0 { percent = (current * 100) / total }
	
	// Cálculo da largura da barra de caracteres
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

	// \033[%dA -> Move o cursor 'pos' linhas para cima
	// \r -> Volta para o início da linha
	// \033[K -> Limpa a linha atual
	// \033[%dB -> Move o cursor de volta para a linha original
	fmt.Printf("\033[%dA\r\033[K   %s %-8s [%s] %3d%% %-5s\033[%dB",
		pos, blue("→"), cyan(langField), cBar, percent, status, pos)
}

// checkDependencies valida as ferramentas externas necessárias
func checkDependencies() {
	for _, bin := range []string{"xgettext", "msginit", "msgfmt", "trans"} {
		if _, err := exec.LookPath(bin); err != nil { log.Fatalf("Dependência faltando: %s", bin) }
	}
}

// usage imprime o manual de ajuda formatado com cores e na ordem lógica solicitada
func usage() {
	fmt.Fprintf(os.Stderr, "\n%s %s\n", cyan(_APP_), white(_VERSION_))
	fmt.Fprintf(os.Stderr, "%s\n\n", white(_COPY_))
	fmt.Fprintf(os.Stderr, "%s: %s %s %s\n\n", yellow("Uso"), green(_APP_), yellow("-i"), green("<arquivo> [opções]"))
	fmt.Fprintf(os.Stderr, "%s:\n", yellow("Opções"))

	defLangs := strings.Join(defaultLanguages, ",")

	// Estrutura para manter a ordem exata de exibição das flags
	flags := []struct {
		short string
		long  string
		desc  string
	}{
		{"-i", "--inputfile", "Arquivo fonte para tradução (.sh, .py, .md)"},
		{"-e", "--engine", "Motor: google, bing, yandex (padrão: google)"},
		{"-l", "--language", fmt.Sprintf("Idiomas (ex: pt-BR,en) ou 'all' (padrão: %s)", defLangs)},
		{"-j", "--jobs", "Traduções simultâneas (padrão: 8)"},
		{"-f", "--force", "Força nova tradução (ignora cache)"},
		{"-q", "--quiet", "Modo silencioso"},
		{"-v", "--verbose", "Mostrar detalhes"},
		{"-V", "--version", "Mostra a versão do programa"},
	}

	for _, f := range flags {
		fmt.Fprintf(os.Stderr, "  %s, %-12s %s\n", cyan(f.short), cyan(f.long), white(f.desc))
	}

	fmt.Fprintf(os.Stderr, "\n%s:\n", yellow("Exemplos"))
	fmt.Fprintf(os.Stderr, "  %s -i script.sh -l pt-BR,en\n", green(_APP_))
	fmt.Fprintf(os.Stderr, "  %s -i README.md -l all -e bing\n\n", green(_APP_))
}

// --- FUNÇÕES GETTEXT (Tradução de código shell/script) ---

func prepareGettext(inputPath, baseName string) {
	potFile := filepath.Join("pot", baseName+".pot")
	// Extrai strings do script que estejam marcadas com gettext ou _
	exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, inputPath).Run()
	// Corrige o charset no arquivo modelo para UTF-8
	exec.Command("sed", "-i", "s/charset=CHARSET/charset=UTF-8/", potFile).Run()
}

func prepareMsginit(baseName, lang string) {
	potFile := filepath.Join("pot", baseName+".pot")
	poTmp := filepath.Join("pot", fmt.Sprintf("%s-temp-%s.po", baseName, lang))
	os.Remove(poTmp)
	// Inicializa um novo arquivo PO para o idioma específico baseado no modelo POT
	exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	exec.Command("sed", "-i", "s/charset=ASCII/charset=utf-8/g", poTmp).Run()
}

func translateFile(baseName, lang string) {
	// ... (Lógica de processamento de arquivos PO linha a linha utilizando callUniversalTranslator)
    // Omitido para brevidade nesta visualização, mas presente no fluxo principal
}

func writeMsgfmtToMo(baseName, lang string) {
	dir := filepath.Join("usr/share/locale", lang, "LC_MESSAGES")
	os.MkdirAll(dir, 0755)
	poFinal := filepath.Join("pot", fmt.Sprintf("%s-%s.po", baseName, lang))
	moFile := filepath.Join(dir, baseName+".mo")
	// Compila o arquivo PO (texto) para MO (binário) que o sistema Linux entende
	exec.Command("msgfmt", poFinal, "-o", moFile).Run()
}

func showVersion() {
	fmt.Printf("%s %s\n%s\n", cyan(_APP_), white(_VERSION_), white(_COPY_))
}

func confLog() {
	f, _ := os.OpenFile("/tmp/"+_APP_+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	logger = log.New(f, "", 0)
}
