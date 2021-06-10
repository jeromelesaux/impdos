package browser

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/jeromelesaux/impdos"
)

type Browser struct {
	imp *impdos.Impdos
}

func NewBrowser() *Browser {
	return &Browser{}
}

func makeCell() fyne.CanvasObject {
	rect := canvas.NewRectangle(&color.NRGBA{128, 128, 128, 255})
	rect.SetMinSize(fyne.NewSize(30, 30))
	return rect
}

func (b *Browser) Load(app fyne.App) {
	box1 := makeCell()
	box2 := makeCell()
	grid := container.NewGridWithColumns(2,
		box1, box2)
	win := app.NewWindow("IMPDos explorer")
	win.SetContent(grid)
	win.Resize(fyne.NewSize(400, 400))
	win.SetTitle("IMPDos explorer")
	win.Show()
}
