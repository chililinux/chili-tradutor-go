package main

import (
	"github.com/gotk3/gotk3/gtk"
	"log"
)

func main() {
	gtk.Init(nil)

	// Criamos o Builder (ele lê o XML do Glade)
	builder, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal("Erro ao criar builder:", err)
	}

	// Carrega o arquivo de interface
	err = builder.AddFromFile("ui.glade")
	if err != nil {
		log.Fatal("Erro ao carregar arquivo .glade:", err)
	}

	// Mapeia os objetos do XML para variáveis Go
	obj, _ := builder.GetObject("main_window")
	win := obj.(*gtk.Window)
	win.Connect("destroy", gtk.MainQuit)

	// Exemplo: Pegar um botão chamado "btn_traduzir" no Glade
	obj, _ = builder.GetObject("btn_traduzir")
	btn := obj.(*gtk.Button)

	btn.Connect("clicked", func() {
		// Aqui entra a sua lógica de tradução v1.7.5!
		println("Iniciando tradução via GUI no Wayland...")
	})

	win.ShowAll()
	gtk.Main()
}
