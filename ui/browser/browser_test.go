package browser

import (
	"fmt"
	"testing"

	"github.com/jeromelesaux/impdos"
)

func TestTreeSearch(t *testing.T) {
	imp := impdos.NewImpdos()
	imp.Partitions = append(imp.Partitions, impdos.NewPartition(0))
	imp.Partitions[0].Inode = GenerateDir("ROOTDIR", imp.Partitions[0], true)
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("FILE%d", i)
		imp.Partitions[0].Inode.Inodes = append(imp.Partitions[0].Inode.Inodes, GenerateFile(name, imp.Partitions[0].Inode))
	}
	c := imp.GetTreePath()
	if len(c) != 3 {
		t.Fatal()
	}

	node := imp.Partitions[0].Inode.Inodes[3]
	n := imp.Partitions[0].Inode.GetInode(node.Uuid)
	if n != node {
		t.Fatal()
	}
	t.Log(imp)
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
