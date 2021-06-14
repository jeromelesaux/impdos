package browser

import (
	"fmt"

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

func GenerateFS(partition *impdos.Partition) *impdos.Inode {
	root := GenerateDir("ROOTDIR", partition, true)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("FILE%d", i)
		root.Inodes = append(root.Inodes, GenerateFile(name, root))
	}
	subFs := GenerateDir("SUBDIR", partition, true)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("FILE%d", i)
		subFs.Inodes = append(subFs.Inodes, GenerateFile(name, subFs))
	}
	subFs2 := GenerateDir("SUBDIR2", partition, true)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("FILET%d", i)
		subFs2.Inodes = append(subFs2.Inodes, GenerateFile(name, subFs2))
	}
	root.Inodes = append(root.Inodes, subFs)
	root.Inodes = append(root.Inodes, subFs2)
	return root
}

func GenerateDir(name string, partition *impdos.Partition, isRoot bool) *impdos.Inode {
	node := impdos.NewInode(0, nil, partition)
	node.IsRoot = isRoot
	node.Type = impdos.DirectoryType
	node.Name = []byte(name)
	return node
}

func GenerateFile(name string, folder *impdos.Inode) *impdos.Inode {
	node := impdos.NewInode(0, folder, folder.Partition)
	node.Type = impdos.FileType
	node.Name = []byte(name)
	node.Size = 16000
	return node
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
		fmt.Printf("Tree node selected: %s", id)
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
