package browser

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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

func (b *Browser) LoadDom(device string) {
	var err error
	b.imp, err = impdos.Read(device)
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
	}
	err = b.imp.ReadCatalogues()
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
	}
	b.treeData = b.imp.GetTreePath()

	b.treeView = widget.NewTreeWithStrings(b.treeData)
	b.treeView.Refresh()
}

func (b *Browser) Load(app fyne.App) {
	b.LoadDom("/Users/jeromelesaux/Downloads/impdos_master_dump.img")
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

	// chemin du path du device
	deviceForm := widget.NewEntry()
	deviceForm.SetText("Device path:")
	deviceForm.OnSubmitted = func(v string) {}
	// bouton pour accéder au path du device
	openDeviceButton := widget.NewButtonWithIcon("Open device", theme.FileIcon(), func() {
		// here
	})

	deviceW := container.NewGridWithRows(2,
		deviceForm,
		openDeviceButton)

	modeLabel := widget.NewLabel("Mode")
	modeValue := widget.NewEntry()
	modeContainer := container.NewGridWithColumns(2,
		modeLabel, modeValue)

	borderLabel := widget.NewLabel("Border")
	borderValue := widget.NewEntry()
	borderContainer := container.NewGridWithColumns(2,
		borderLabel,
		borderValue)

	paperLabel := widget.NewLabel("Paper")
	paperValue := widget.NewEntry()
	paperContainer := container.NewGridWithColumns(2,
		paperLabel,
		paperValue)

	inkLabel := widget.NewLabel("Ink")
	inkValue := widget.NewEntry()
	inkContainer := container.NewGridWithColumns(2,
		inkLabel,
		inkValue)

	autoexecButton := widget.NewButton("Apply autoexec.", func() {})

	autoexecContainer := container.NewGridWithRows(5,
		modeContainer,
		borderContainer,
		paperContainer,
		inkContainer,
		autoexecButton)

	backupButton := widget.NewButton("Backup your ImpDOS DOM", func() {})
	restoreButton := widget.NewButton("Restore your ImpDOS DOM", func() {})
	extractButton := widget.NewButton("Extract files or folder from you ImpDOS DOM", func() {})
	importButton := widget.NewButton("Import your files or folder to your ImpDOS DOM", func() {})
	domActionsContainer := container.NewGridWithRows(4,
		backupButton,
		restoreButton,
		extractButton,
		importButton)
	cmdContainer := container.NewGridWithRows(6,
		deviceW,
		autoexecContainer,
		domActionsContainer,
	)

	grid := container.NewGridWithColumns(2,
		b.treeView,
		cmdContainer)

	win := app.NewWindow("IMPDos explorer")
	win.SetContent(grid)
	win.Resize(fyne.NewSize(1200, 800))
	win.SetTitle("IMPDos explorer")
	win.Show()
}
