package main

import (
	"github.com/gotk3/gotk3/gtk"
	"log"
)

func main() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("Chili Installer GUI")
	win.Connect("destroy", func() {
		gtk.MainQuit()
	})

	btn, _ := gtk.ButtonNewWithLabel("Clique aqui para Traduzir")
	btn.Connect("clicked", func() {
		log.Println("Botão pressionado!")
	})

	win.Add(btn)
	win.ShowAll()
	gtk.Main()
}
