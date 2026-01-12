package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

func main() {
	// Inicializa o GTK
	gtk.Init(nil)

	// Criar Janela Principal
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("Chili Tradutor GUI")
	win.SetDefaultSize(500, 300)
	win.SetPosition(gtk.WIN_POS_CENTER)
	win.Connect("destroy", gtk.MainQuit)

	// Layout Vertical
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginTop(20)
	win.Add(box)

	// Label de Título
	lbl, _ := gtk.LabelNew("Selecione o arquivo para traduzir")
	box.PackStart(lbl, false, false, 5)

	// Entrada de Texto (Caminho do Arquivo)
	entry, _ := gtk.EntryNew()
	entry.SetPlaceholderText("/caminho/para/o/arquivo")
	box.PackStart(entry, false, false, 5)

	// Botão de Seleção de Arquivo
	btnFile, _ := gtk.ButtonNewWithLabel("Procurar Arquivo...")
	box.PackStart(btnFile, false, false, 5)

	// Barra de Progresso
	pbar, _ := gtk.ProgressBarNew()
	pbar.SetShowText(true)
	box.PackStart(pbar, false, false, 10)

	// Botão Iniciar
	btnStart, _ := gtk.ButtonNewWithLabel("Iniciar Tradução")
	btnStart.SetSensitive(false)
	box.PackStart(btnStart, false, false, 20)

	// Lógica do Botão Procurar
	btnFile.Connect("clicked", func() {
		fileChooser, _ := gtk.FileChooserDialogNewWith2Buttons(
			"Selecione o arquivo", win, gtk.FILE_CHOOSER_ACTION_OPEN,
			"Cancelar", gtk.RESPONSE_CANCEL, "Abrir", gtk.RESPONSE_ACCEPT,
		)
		if fileChooser.Run() == gtk.RESPONSE_ACCEPT {
			filename := fileChooser.GetFilename()
			entry.SetText(filename)
			btnStart.SetSensitive(true)
		}
		fileChooser.Destroy()
	})

	// Lógica do Botão Iniciar (Aqui chamamos o seu motor de tradução)
	btnStart.Connect("clicked", func() {
		filename := entry.GetText()
		
		// Simulação de Progresso (Isso seria integrado ao loop de tradução)
		go func() {
			for i := 0.0; i <= 1.0; i += 0.1 {
				// No GTK, atualizações de UI devem ser feitas na thread principal
				gtk.IdleAdd(func() {
					pbar.SetFraction(i)
					pbar.SetText(fmt.Sprintf("Traduzindo... %.0f%%", i*100))
				})
				
				// Simulando tempo de tradução
				// Aqui você chamaria o seu translateFile()
				fmt.Println("Processando:", filename)
				// time.Sleep(500 * time.Millisecond)
			}
			
			gtk.IdleAdd(func() {
				pbar.SetText("Concluído!")
				showDialog(win, "Sucesso", "Tradução finalizada com sucesso!")
			})
		}()
	})

	win.ShowAll()
	gtk.Main()
}

// Função auxiliar para diálogos de aviso
func showDialog(parent *gtk.Window, title, msg string) {
	dialog := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_INFO, gtk.BUTTONS_OK, msg)
	dialog.SetTitle(title)
	dialog.Run()
	dialog.Destroy()
}
