package browser

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jeromelesaux/fyne-io/custom_widget"
	"github.com/jeromelesaux/impdos"
	"github.com/jeromelesaux/impdos/ui/usb"
)

var (
	backupFileFilter = storage.NewExtensionFileFilter([]string{".img", ".ibc"})
	ibcFileFilter    = storage.NewExtensionFileFilter([]string{".ibc"})
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

func (b *Browser) ReloadUI() {
	b.LoadDom(b.devicePath.Text)
}

func (b *Browser) LoadDom(device string) {
	var err error
	b.imp, err = impdos.Read(device)
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
		dialog.ShowError(err, b.window)
	}
	err = b.imp.ReadCatalogues()
	if err != nil {
		fmt.Printf("[LOADING] error :%v\n", err)
		dialog.ShowError(err, b.window)
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

func (b *Browser) extractFile(dest string, node *impdos.Inode) error {
	content, err := node.GetFile(b.imp.Pointer)
	if err != nil {
		return err
	}
	fw, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fw.Close()
	_, err = fw.Write(content)
	if err != nil {
		return err
	}
	return nil
}

func (b *Browser) extractFolder(root string, node *impdos.Inode) error {
	for _, next := range node.Inodes {
		if next.IsListable() {
			if next.IsDir() {
				newDest := filepath.Join(root, next.GetName())
				err := os.Mkdir(newDest, os.ModePerm)
				if err != nil {
					return err
				}
				if err := b.extractFolder(newDest, next); err != nil {
					return err
				}
			} else {
				dest := filepath.Join(root, next.GetName())
				if err := b.extractFile(dest, next); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (b *Browser) importFolder(from string, node *impdos.Inode) error {
	files, err := ioutil.ReadDir(from)
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(from, file.Name())
			if err := b.imp.Partitions[node.Partition.PartitionNumber].Save(filePath, b.imp.Pointer, node); err != nil {
				return err
			}
		} else {
			folderPath := filepath.Join(from, file.Name())
			_, err := b.imp.Partitions[node.Partition.PartitionNumber].NewFolder(folderPath, b.imp.Pointer, node)
			if err != nil {
				return err
			}
			if err := b.importFolder(folderPath, node.Inodes[len(node.Inodes)-1]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Browser) Load(app fyne.App) {
	devices, err := usb.DevicesDetect()
	if err != nil {
		devices = []string{}
		fmt.Printf("[UI IMPDOS] error while getting usb devices error : %v\n", err)
	}
	/*	tree.OnUnselected = func(id string) {
			fmt.Printf("Tree node unselected: %s", id)
		}
	*/

	devicesSelect := widget.NewSelect(devices, func(device string) {
		b.LoadDom(device)
	})

	// chemin du path du device
	b.devicePath = widget.NewEntry()
	b.devicePath.SetText("")
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
		fd.SetFilter(backupFileFilter)
		fd.Resize(fyne.NewSize(1000, 800))
		fd.Show()
		// here
	})

	/*	deviceW := container.NewGridWithRows(3,
		devicesSelect,
		b.devicePath,
		openDeviceButton)
	*/
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

	autoexecButton := widget.NewButton("Apply autoexec.", func() {
		var border, ink, paper, mode byte
		fmt.Sscanf(b.border.Text, "%d", &border)
		fmt.Sscanf(b.ink.Text, "%d", &ink)
		fmt.Sscanf(b.paper.Text, "%d", &paper)
		fmt.Sscanf(b.mode.Text, "%d", &mode)
		a := &impdos.AutoExec{
			Border: border,
			Paper:  paper,
			Ink:    ink,
			Mode:   mode,
			End:    0xff,
		}
		if err := b.imp.SaveAutoexec(a); err != nil {
			dialog.ShowError(err, b.window)
			return
		}
	})

	b.treeView = widget.NewTree(nil, nil, nil, nil)

	backupButton := widget.NewButton("Backup your ImpDOS DOM", func() {
		if b.devicePath.Text == "" {
			dialog.ShowError(errors.New("no device selected"), b.window)
			return
		}

		fs := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			if writer == nil {
				return
			}

			backupFile := writer.URI().Path()
			os.Remove(backupFile)
			backupFile += ".ibc"
			np := custom_widget.NewProgressInfinite("Backup DOM to "+backupFile, b.window)
			//			np := dialog.NewProgress("backup your DOM", "Backup DOM to "+backupFile, b.window)
			go func() {
				f, err := os.Create(backupFile)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				defer f.Close()
				buf := make([]byte, 1024)
				fr, err := os.Open(b.devicePath.Text)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				defer fr.Close()
				_, err = fr.Seek(0, io.SeekEnd)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				_, err = fr.Seek(0, io.SeekStart)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				var copied int
				for {
					_, err := fr.Read(buf)
					if err != nil {
						fmt.Printf("[RESTORE BACKUP] error :%v\n", err)
						break
					}
					n, err := f.Write(buf)
					if err != nil {
						dialog.ShowError(err, b.window)
						np.Hide()
						return
					}
					copied += n
					// np.SetValue(float64(copied) / float64(nb))
				}
				np.Hide()
				dialog.ShowInformation("Backup ended", "Your image backup is here"+backupFile, b.window)
			}()
			np.Show()
		}, b.window)
		fs.SetFilter(ibcFileFilter)
		fs.Show()
	})
	restoreButton := widget.NewButton("Restore your ImpDOS DOM", func() {
		if b.devicePath.Text == "" {
			dialog.ShowError(errors.New("no device selected"), b.window)
			return
		}
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			if reader == nil {
				return
			}
			backupFile := reader.URI().Path()
			np := custom_widget.NewProgressInfinite("Backup DOM to "+backupFile, b.window)
			go func() {
				f, err := os.Create(b.devicePath.Text)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				defer f.Close()
				buf := make([]byte, 1024)
				fr, err := os.Open(backupFile)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				defer fr.Close()
				_, err = fr.Seek(0, os.SEEK_END)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				_, err = fr.Seek(0, io.SeekStart)
				if err != nil {
					dialog.ShowError(err, b.window)
					np.Hide()
					return
				}
				var copied int
				for {
					_, err := fr.Read(buf)
					if err != nil {
						fmt.Printf("[RESTORE BACKUP] error :%v\n", err)
						break
					}
					n, err := f.Write(buf)
					if err != nil {
						dialog.ShowError(err, b.window)
						np.Hide()
						return
					}
					copied += n
					//	np.SetValue(float64(copied) / float64(nb))
				}
				np.Hide()
				dialog.ShowInformation("Restoration ended", "Your device is restored with backup image : "+backupFile, b.window)
			}()

			np.Show()
		}, b.window)
	})
	extractButton := widget.NewButton("Extract files or folder from you ImpDOS DOM", func() {
		if b.devicePath.Text == "" {
			dialog.ShowError(errors.New("no device selected"), b.window)
			return
		}
		if b.uuidSelected == "" {
			dialog.ShowError(errors.New("you did not select a destination folder"), b.window)
			return
		}
		node := b.imp.GetInode(b.uuidSelected)
		if node == nil {
			dialog.ShowError(errors.New("can not find the folder"), b.window)
			return
		}
		if node.IsDir() {
			dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				if list == nil {
					return
				}
				root := list.Path()
				root = filepath.Join(root, node.GetName())
				err = os.Mkdir(root, os.ModePerm)
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}

				if err := b.extractFolder(root, node); err != nil {
					dialog.ShowError(err, b.window)
					return
				}
			}, b.window)
		} else {
			dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				if writer == nil {
					return
				}
				dest := writer.URI().Path()
				content, err := node.GetFile(b.imp.Pointer)
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				fw, err := os.Create(dest)
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				defer fw.Close()
				_, err = fw.Write(content)
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
			}, b.window)
		}
	})
	importFolderButton := widget.NewButton("Import a folder to your ImpDOS DOM", func() {
		if b.uuidSelected == "" {
			dialog.ShowError(errors.New("you did not select a folder"), b.window)
			return
		}
		node := b.imp.GetInode(b.uuidSelected)
		if node == nil {
			dialog.ShowError(errors.New("can not find the folder"), b.window)
			return
		}
		dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			if list == nil {
				return
			}
			root := list.Path()
			rootName := filepath.Base(root)
			newInode, err := b.imp.Partitions[node.Partition.PartitionNumber].NewFolder(rootName, b.imp.Pointer, node)
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}

			if err := b.importFolder(root, newInode); err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			b.ReloadUI()
		}, b.window)
	})
	importFileButton := widget.NewButton("Import a file to your ImpDOS DOM", func() {
		if b.uuidSelected == "" {
			dialog.ShowError(errors.New("you did not select a folder"), b.window)
			return
		}
		node := b.imp.GetInode(b.uuidSelected)
		if node == nil {
			dialog.ShowError(errors.New("can not find the folder"), b.window)
			return
		}
		dialog.ShowFileOpen(func(writer fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			if writer == nil {
				return
			}
			filePath := writer.URI().Path()
			if err := b.imp.Partitions[node.Partition.PartitionNumber].Save(filePath, b.imp.Pointer, node); err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			b.ReloadUI()
		}, b.window)
	})
	deleteNode := widget.NewButton("Delete the selected file or folder", func() {
		if b.uuidSelected == "" {
			dialog.ShowError(errors.New("you did not select a folder"), b.window)
			return
		}
		node := b.imp.GetInode(b.uuidSelected)
		if node == nil {
			dialog.ShowError(errors.New("can not find the folder"), b.window)
			return
		}
		dialog.ShowConfirm("Delete Folder",
			"confirm your choice.",
			func(confirm bool) {
				if err := node.Delete(); err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				pn := node.Partition.PartitionNumber
				folder := node.Previous
				if err := b.imp.Partitions[pn].SaveInodeEntry(b.imp.Pointer, folder, node); err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				b.ReloadUI()
			},
			b.window)
	})
	createFolder := widget.NewButton("Create new folder", func() {
		if b.uuidSelected == "" {
			dialog.ShowError(errors.New("you did not select a folder"), b.window)
			return
		}
		node := b.imp.GetInode(b.uuidSelected)
		if node == nil {
			dialog.ShowError(errors.New("can not find the folder"), b.window)
			return
		}
		dialog.ShowEntryDialog("Please choose a folder name",
			"Will create a new folder on your DOM",
			func(ok string) {
				pn := node.Partition.PartitionNumber
				_, err := b.imp.Partitions[pn].NewFolder(ok, b.imp.Pointer, node)
				if err != nil {
					dialog.ShowError(err, b.window)
					return
				}
				b.ReloadUI()
			},
			b.window)
	})
	autoexecInfo := container.NewGridWithColumns(2,
		modeContainer,
		borderContainer,
		paperContainer,
		inkContainer,
	)
	autoexecPanel := container.NewGridWithRows(5,
		devicesSelect,
		b.devicePath,
		openDeviceButton,
		autoexecInfo,
		autoexecButton,
	)
	actionsContainer := container.NewGridWithRows(7,
		backupButton,
		restoreButton,
		extractButton,
		importFolderButton,
		importFileButton,
		deleteNode,
		createFolder,
	)
	cmdContainer := container.NewGridWithRows(2,
		autoexecPanel,
		actionsContainer,
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
