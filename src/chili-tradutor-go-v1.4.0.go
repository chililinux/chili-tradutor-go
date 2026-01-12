/*
   chili-tradutor-go - v1.4.1
   Copyright (c) 2023-2026, Vilmar Catafesta <vcatafesta@gmail.com>
*/

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/pflag"
)

const (
	_APP_     = "chili-tradutor-go"
	_VERSION_ = "1.4.1-20260110"
	_COPY_    = "Copyright (C) 2023-2026 Vilmar Catafesta, <vcatafesta@gmail.com>"
)

var (
	cyan    = color.New(color.Bold, color.FgCyan).SprintFunc()
	orange  = color.New(color.FgYellow).SprintFunc()
	yellow  = color.New(color.Bold, color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	black   = color.New(color.Bold, color.FgBlack).SprintFunc()
)

var (
	inputFile    string
	engine       string // Nova variável para o motor
	showVersion  bool
	showHelp     bool
	forceFlag    bool
	quietFlag    bool
	nocolorFlag  bool
	languages    []string
	logger       *log.Logger
)

var supportedLanguages = []string{
	"ar", "bg", "cs", "da", "de", "el", "en", "es", "et",
	"fa", "fi", "fr", "he", "hi", "hr", "hu", "is", "it",
	"ja", "ko", "nl", "no", "pl", "pt-PT", "pt-BR", "ro",
	"ru", "sk", "sv", "tr", "uk", "zh",
}

func confLog() {
	fileLog := "/tmp/" + _APP_ + ".log"
	logFile, err := os.OpenFile(fileLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir o arquivo de log: %v", err)
	}
	if quietFlag {
		logger = log.New(logFile, "", 0)
	} else {
		logger = log.New(io.MultiWriter(os.Stdout, logFile), "", 0)
	}
}

func checkDependencies() {
	required := []string{"xgettext", "msginit", "msgfmt", "sed", "trans"}
	for _, bin := range required {
		if _, err := exec.LookPath(bin); err != nil {
			log.Fatalf("%s Dependência faltando: %s", red("[ERROR]"), bin)
		}
	}
}

// Função para validar se o motor e a internet estão funcionando
func checkEngineHealth(eng string) error {
	logger.Printf("%s Verificando motor [%s]...", black("[CHECK]"), cyan(eng))
	cmd := exec.Command("trans", "-e", eng, "-no-autocorrect", "-b", ":en", "saúde")
	output, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("motor %s não está respondendo ou sem conexão", eng)
	}
	return nil
}

func main() {
	checkDependencies()

	pflag.StringVarP(&inputFile, "inputfile", "i", "", "Arquivo de entrada")
	pflag.StringVarP(&engine, "engine", "e", "google", "Motor de tradução (google, bing, yandex, apertium)")
	pflag.BoolVarP(&showVersion, "version", "V", false, "Mostrar versão")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Mostrar help")
	pflag.BoolVarP(&forceFlag, "force", "f", false, "Forçar tradução")
	pflag.BoolVarP(&quietFlag, "quiet", "q", false, "Modo quieto")
	pflag.BoolVarP(&nocolorFlag, "nocolor", "n", false, "Sem cores")
	pflag.StringSliceVarP(&languages, "language", "l", nil, "Idiomas destino")

	pflag.Parse()

	if showVersion {
		fmt.Printf("%s versão %s\n", _APP_, _VERSION_)
		os.Exit(0)
	}
	if showHelp || inputFile == "" {
		usage()
		os.Exit(0)
	}
	if nocolorFlag {
		color.NoColor = true
	}

	confLog()

	// Valida o motor antes de começar
	if err := checkEngineHealth(engine); err != nil {
		logger.Fatalf("%s Erro Crítico: %v", red("[ERROR]"), err)
	}

	if len(languages) > 0 {
		supportedLanguages = languages
	}

	prepareGettext(inputFile)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for _, lang := range supportedLanguages {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			prepareMsginit(inputFile, l)
			translateFile(inputFile, l, engine) // Passa o motor
			writeMsgfmtToMo(inputFile, l)
			os.Remove(fmt.Sprintf("%s-temp-%s.po", inputFile, l))
		}(lang)
	}
	wg.Wait()
	logger.Printf("%s Tradução finalizada com motor: %s", green("[SUCESSO]"), engine)
}

func prepareGettext(inputFile string) {
	potFile := inputFile + ".pot"
	if _, err := os.Stat(potFile); os.IsNotExist(err) || forceFlag {
		logger.Printf("%s Extraindo strings: %s", black("[XGETTEXT]"), magenta(inputFile))
		cmd := exec.Command("xgettext", "--from-code=UTF-8", "--language=shell", "--keyword=gettext", "--keyword=_", "--output="+potFile, inputFile)
		_ = cmd.Run()
	}
	exec.Command("sed", "-i", `s/charset=CHARSET/charset=UTF-8/`, potFile).Run()
}

func prepareMsginit(inputFile, lang string) {
	potFile := inputFile + ".pot"
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)
	logger.Printf("%s Inicializando [%s]", black("[MSGINIT]"), cyan(lang))
	_ = exec.Command("msginit", "--no-translator", "--locale="+lang, "--input="+potFile, "--output="+poTmp).Run()
	exec.Command("sed", "-i", `s/charset=ASCII/charset=utf-8/g`, poTmp).Run()
}

func translateFile(inputFile, lang, eng string) {
	poFinal := fmt.Sprintf("%s-%s.po", inputFile, lang)
	poTmp := fmt.Sprintf("%s-temp-%s.po", inputFile, lang)

	if _, err := os.Stat(poFinal); !os.IsNotExist(err) && !forceFlag {
		return
	}

	inFile, _ := os.Open(poTmp)
	defer inFile.Close()
	outFile, _ := os.Create(poFinal)
	defer outFile.Close()

	scanner := bufio.NewScanner(inFile)
	var msgidLines []string
	var isMsgid bool

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid ") {
			isMsgid = true
			msgidLines = []string{strings.TrimPrefix(line, "msgid ")}
			continue
		}
		if strings.HasPrefix(line, "msgstr ") && isMsgid {
			fullMsgid := strings.Join(msgidLines, " ")
			msgstr := translateMessage(fullMsgid, lang, eng) // Usa motor
			fmt.Fprintf(outFile, "msgid %s\n%s\n", strings.Join(msgidLines, "\n"), msgstr)
			isMsgid = false
			msgidLines = nil
			continue
		}
		if isMsgid {
			msgidLines = append(msgidLines, line)
		} else {
			fmt.Fprintln(outFile, line)
		}
	}
}

func translateMessage(msgid, lang, eng string) string {
	msgid = strings.TrimSpace(msgid)
	if msgid == "" || msgid == `""` {
		return "msgstr \"\""
	}

	cleanMsgid := strings.Trim(msgid, `"`)
	// Adicionado parâmetro -e para selecionar o motor
	cmd := exec.Command("trans", "-e", eng, "-no-autocorrect", "-b", ":"+lang, cleanMsgid)
	output, err := cmd.Output()

	result := cleanMsgid
	if err == nil {
		result = strings.TrimSpace(string(output))
		result = strings.ReplaceAll(result, `"`, `\"`)
	}

	lines := strings.Split(result, "\n")
	if len(lines) == 1 {
		return fmt.Sprintf("msgstr \"%s\"", lines[0])
	}

	formatted := "msgstr \"\"\n"
	for i, l := range lines {
		suffix := "\\n"
		if i == len(lines)-1 {
			suffix = ""
		}
		formatted += fmt.Sprintf("\"%s%s\"\n", l, suffix)
	}
	return formatted
}

func writeMsgfmtToMo(inputFile, lang string) {
	dir := "usr/share/locale/" + lang + "/LC_MESSAGES"
	os.MkdirAll(dir, 0755)
	poFile := fmt.Sprintf("%s-%s.po", inputFile, lang)
	moFile := fmt.Sprintf("%s/%s.mo", dir, inputFile)
	_ = exec.Command("msgfmt", poFile, "-o", moFile).Run()
	logger.Printf("%s Gerado: %s", green("[OK]"), moFile)
}

func usage() {
	fmt.Printf("\n%s\n%s\n\n", yellow(_APP_), _COPY_)
	fmt.Println("Uso: chili-tradutor-go -i <arquivo> [opções]")
	fmt.Println("\nOpções:")
	fmt.Println("  -i, --inputfile   Arquivo fonte .sh")
	fmt.Println("  -e, --engine      Motor: google (default), bing, yandex")
	fmt.Println("  -l, --language    Idiomas (ex: en,es,fr)")
	fmt.Println("  -f, --force       Força nova tradução")
	fmt.Println("  -q, --quiet       Saída silenciosa")
	fmt.Println("  -h, --help        Ajuda")
}
