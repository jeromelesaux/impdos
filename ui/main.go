package main

import (
	"os"

	"fyne.io/fyne/v2/app"
	"github.com/jeromelesaux/impdos/ui/browser"
)

func main() {
	os.Setenv("FYNE_SCALE", "0.6")
	/* main application */
	app := app.NewWithID("IMPDos Explorer")
	browser := browser.NewBrowser()
	browser.Load(app)
	app.Run()
}
