package browser

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jeromelesaux/impdos"
)

type Browser struct {
	imp          *impdos.Impdos
	treeData     map[string][]string
	treeView     *widget.Tree
	paper        *widget.Entry
	border       *widget.Entry
	ink          *widget.Entry
	mode         *widget.Entry
	devicePath   *widget.Entry
	window       fyne.Window
	treeViewGrid *fyne.Container
	uuidSelected string
}

func (b *Browser) updateUi() {
	if b.imp != nil {
		v, err := b.imp.ReadAutoExec()
		if err != nil {
			fmt.Printf("[UPDATE UI AUTOEXEC] cannot update ui error : %v\n", err)
			return
		}

		b.paper.SetText(fmt.Sprintf("%d", v.Paper))
		b.border.SetText(fmt.Sprintf("%d", v.Border))
		b.ink.SetText(fmt.Sprintf("%d", v.Ink))
		b.mode.SetText(fmt.Sprintf("%d", v.Mode))

	}
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
	b.devicePath.SetText(device)
	b.updateUi()
	b.treeData = b.imp.GetTreePath()
	b.treeView = widget.NewTree(
		func(uid string) (c []string) {
			c = b.treeData[uid]
			return
		},
		func(uid string) (ok bool) {
			_, ok = b.treeData[uid]
			return
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Object")
		},
		func(uid string, branch bool, node fyne.CanvasObject) {
			node.(*widget.Label).SetText(uid)
		},
	)
	b.treeView.OnSelected = func(id string) {
		start := strings.Index(id, "(")
		end := strings.LastIndex(id, ")")

		var uuid string
		if start >= 0 && end >= 0 {
			uuid = id[start+1 : end]
			b.uuidSelected = uuid
		}
		fmt.Printf("Tree node selected: %s with uuid :%s\n", id, uuid)
	}
	b.treeViewGrid.Objects[0] = b.treeView
	b.treeView.Refresh()
}

func (b *Browser) Load(app fyne.App) {

	/*	tree.OnUnselected = func(id string) {
			fmt.Printf("Tree node unselected: %s", id)
		}
	*/

	// chemin du path du device
	b.devicePath = widget.NewEntry()
	b.devicePath.SetText("Device path:")
	b.devicePath.OnSubmitted = func(v string) {
		var err error
		b.imp, err = impdos.Read(v)
		if err != nil {
			fmt.Printf("[DEVICE LOADING] cannot load device error : %v\n", err)
			return
		}
	}
	// bouton pour accéder au path du device
	openDeviceButton := widget.NewButtonWithIcon("Open device", theme.FileIcon(), func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader == nil {
				return
			}
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			filename := reader.URI().Path()
			b.LoadDom(filename)
		}, b.window)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".img", ".ibc"}))
		fd.Resize(fyne.NewSize(1000, 800))
		fd.Show()
		// here
	})

	deviceW := container.NewGridWithRows(2,
		b.devicePath,
		openDeviceButton)

	modeLabel := widget.NewLabel("Mode")
	b.mode = widget.NewEntry()
	modeContainer := container.NewGridWithColumns(2,
		modeLabel,
		b.mode,
	)

	borderLabel := widget.NewLabel("Border")
	b.border = widget.NewEntry()
	borderContainer := container.NewGridWithColumns(2,
		borderLabel,
		b.border,
	)

	paperLabel := widget.NewLabel("Paper")
	b.paper = widget.NewEntry()
	paperContainer := container.NewGridWithColumns(2,
		paperLabel,
		b.paper,
	)

	inkLabel := widget.NewLabel("Ink")
	b.ink = widget.NewEntry()
	inkContainer := container.NewGridWithColumns(2,
		inkLabel,
		b.ink,
	)

	autoexecButton := widget.NewButton("Apply autoexec.", func() {})

	autoexecContainer := container.NewGridWithRows(5,
		modeContainer,
		borderContainer,
		paperContainer,
		inkContainer,
		autoexecButton)

	b.treeView = widget.NewTree(nil, nil, nil, nil)

	backupButton := widget.NewButton("Backup your ImpDOS DOM", func() {})
	restoreButton := widget.NewButton("Restore your ImpDOS DOM", func() {})
	extractButton := widget.NewButton("Extract files or folder from you ImpDOS DOM", func() {})
	importButton := widget.NewButton("Import your files or folder to your ImpDOS DOM", func() {})
	deleteNode := widget.NewButton("Delete the selected file or folder", func() {})
	createFolder := widget.NewButton("Create new folder", func() {})

	domActionsContainer := container.NewGridWithRows(6,
		backupButton,
		restoreButton,
		extractButton,
		importButton,
		deleteNode,
		createFolder)

	cmdContainer := container.NewGridWithRows(3,
		deviceW,
		autoexecContainer,
		domActionsContainer,
	)

	b.treeViewGrid = container.NewGridWithColumns(2,
		b.treeView,
		cmdContainer)

	b.window = app.NewWindow("IMPDos explorer")
	b.window.SetContent(b.treeViewGrid)
	b.window.Resize(fyne.NewSize(1200, 800))
	b.window.SetTitle("IMPDos explorer")
	b.window.Show()
}
