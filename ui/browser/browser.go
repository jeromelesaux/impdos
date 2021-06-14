package browser

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/jeromelesaux/impdos"
)

type Browser struct {
	imp      *impdos.Impdos
	treeData map[string][]string
	treeView *widget.Tree
}

func NewBrowser() *Browser {
	return &Browser{}
}

func (b *Browser) Load(app fyne.App) {
	var err error
	b.imp, err = impdos.Read("/Users/jeromelesaux/Downloads/impdos_master_dump.img")
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
	}
	err = b.imp.ReadCatalogues()
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
	}
	b.treeData = b.imp.GetTreePath()

	b.treeView = widget.NewTreeWithStrings(b.treeData)
	b.treeView.OnSelected = func(id string) {
		start := strings.Index(id, "(")
		end := strings.LastIndex(id, ")")

		var uuid string
		if start >= 0 && end >= 0 {
			uuid = id[start+1 : end]
		}
		fmt.Printf("Tree node selected: %s with uuid :%s\n", id, uuid)
	}
	/*	tree.OnUnselected = func(id string) {
			fmt.Printf("Tree node unselected: %s", id)
		}
	*/
	grid := container.NewGridWithColumns(1,
		b.treeView)
	win := app.NewWindow("IMPDos explorer")
	win.SetContent(grid)
	win.Resize(fyne.NewSize(1000, 800))
	win.SetTitle("IMPDos explorer")
	win.Show()
}
