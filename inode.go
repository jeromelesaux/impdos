package impdos

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
)

var (
	DirectoryType   byte = 0x10
	FileType        byte = 0
	UnusedConstant0      = []byte{0x0, 0x0, 0x5d, 0x0, 0x21, 0x36, 0x0, 0x0, 0x0, 0x0, 0x5d, 0x0, 0x21, 0x36}
)

type Inode struct {
	Name            []byte
	Type            byte   // 0 is a file #10 is a directory
	Unused          []byte // constant
	Cluster         uint16 // cluster number
	Size            uint32
	Inodes          []*Inode // files in the directory
	PartitionOffset int64
	IsRoot          bool
	Uuid            string
	Previous        *Inode
	Partition       *Partition
}

func (i *Inode) Path(k string, c map[string][]string) map[string][]string {
	if !i.IsDir() {
		path := fmt.Sprintf("%s\t%d ko\t(%s)", i.GetName(), (i.Size / 1024), i.Uuid)
		c[k] = append(c[k], path)
		return c
	}
	if i.IsDir() && i.IsListable() {
		dir := make([]string, 0)
		name := fmt.Sprintf("%s (%s)", i.GetName(), i.Uuid)
		c[k] = append(c[k], name)
		c[name] = dir
		for _, v := range i.Inodes {
			c = v.Path(name, c)
		}
	}
	return c
}

func (i *Inode) GetInode(uuid string) *Inode {
	if i.Uuid == uuid {
		return i
	}
	for _, v := range i.Inodes {
		if in := v.GetInode(uuid); in != nil {
			return in
		}
	}
	return nil
}

func (i *Inode) ListCatalogue(space string) string {
	var c string
	c += fmt.Sprintf("%s[%.8s]\n", space, string(i.Name))
	for _, v := range i.Inodes {

		if v.IsDir() {
			//c += fmt.Sprintf("%s[%.8s]\n", space, string(v.Name))
			c += v.ListCatalogue(space + "-")
		} else {
			c += fmt.Sprintf("%s%.8s %.4d Ko\n", space, string(v.Name), v.Size/1000)
		}
	}
	return c
}

func (i *Inode) GetHighestN() uint16 {
	var n uint16

	for _, v := range i.Inodes {
		if v.Cluster > n {
			n = v.Cluster
		}
		if v.IsDir() {
			vv := v.GetHighestN()
			if vv > n {
				n = vv
			}
		}
	}

	return n
}

func (in *Inode) GetName() string {
	var s string
	for i := 0; i < len(in.Name); i++ {
		var c byte = 32
		if in.Name[i] >= 48 && in.Name[i] <= 57 {
			c = in.Name[i]
		}
		if in.Name[i] >= 65 && in.Name[i] <= 90 {
			c = in.Name[i]
		}
		if in.Name[i] >= 97 && in.Name[i] <= 122 {
			c = in.Name[i]
		}
		if in.Name[i] == 46 {
			c = in.Name[i]
		}

		s += string(c)
	}
	return s
}

func (i *Inode) Save(f *os.File) error {
	if err := binary.Write(f, binary.BigEndian, i.Name); err != nil {
		return err
	}
	if err := binary.Write(f, binary.BigEndian, &i.Type); err != nil {
		return err
	}
	if err := binary.Write(f, binary.BigEndian, i.Unused); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, &i.Cluster); err != nil {
		return err
	}
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, i.Size)
	if err := binary.Write(f, binary.LittleEndian, size); err != nil {
		return err
	}
	return nil
}

func (i *Inode) Read(f *os.File) error {
	if err := binary.Read(f, binary.BigEndian, i.Name); err != nil {
		return err
	}
	if err := binary.Read(f, binary.BigEndian, &i.Type); err != nil {
		return err
	}
	if err := binary.Read(f, binary.BigEndian, i.Unused); err != nil {
		return err
	}
	if err := binary.Read(f, binary.LittleEndian, &i.Cluster); err != nil {
		return err
	}
	size := make([]byte, 4)
	if err := binary.Read(f, binary.LittleEndian, size); err != nil {
		return err
	}
	i.Size = binary.LittleEndian.Uint32(size)

	return nil
}

func (i *Inode) ReadCatalogue(f *os.File) error {
	for {
		inode := NewInode(i.PartitionOffset, i, i.Partition)
		if err := inode.Read(f); err != nil {
			return err
		}
		if inode.IsEnd() {
			break
		}
		if inode.Type == DirectoryType && inode.Name[0] != '.' && inode.Name[1] != '.' {
			if isPrint(inode.Name) {
				offset, err := f.Seek(0, io.SeekCurrent)
				if err != nil {
					return err
				}
				nextCatalogueOffset := inode.ClusterOffset()

				fmt.Printf("Name:%s Offset :%x next catalogue offset :%x\n",
					string(inode.Name),
					offset,
					nextCatalogueOffset)
				_, err = f.Seek(int64(nextCatalogueOffset), io.SeekStart)
				if err != nil {
					return err
				}
				if err = inode.ReadCatalogue(f); err != nil {
					return err
				}
				_, err = f.Seek(int64(offset), io.SeekStart) // return to initial offset
				if err != nil {
					return err
				}
			}
		}
		i.Inodes = append(i.Inodes, inode)
	}
	return nil
}

func (i *Inode) IsEnd() bool {
	if i.Name[0] == 0xE {
		return true
	}
	return i.Type == EndOfCatalogueType
}

func NewInode(partitionOffset int64, previous *Inode, partition *Partition) *Inode {
	return &Inode{
		Name:            make([]byte, 11),
		Unused:          make([]byte, 14),
		Inodes:          make([]*Inode, 0),
		PartitionOffset: partitionOffset,
		Uuid:            uuid.New().String(),
		Previous:        previous,
		Partition:       partition,
	}
}

func InitInode(partitionOffset int64, previous *Inode, partition *Partition, cluster uint16, size uint32, inodeType byte, name []byte) *Inode {
	inode := NewInode(partitionOffset, previous, partition)
	inode.Cluster = cluster
	inode.Size = size
	inode.Type = inodeType
	copy(inode.Name, name)
	copy(inode.Unused, UnusedConstant0)
	return inode
}

func (in *Inode) IsDir() bool {
	return in.Type == DirectoryType
}

func (i *Inode) IsListable() bool {
	if i.Name[0] == 0xE5 {
		return false
	}
	if i.Name[0] == 46 {
		return false
	}
	if string(i.Name) == "TRASH      " {
		return false
	}
	return true
}

// secteur du catalgoue root toujours en secteur 201 soit offset 0x200*0x201 (512*513) 262656

func (i *Inode) FindInode(name []byte) *Inode {
	toSearch := strings.Trim(string(name), " ")
	for _, v := range i.Inodes {
		v1 := strings.Trim(string(v.Name), " ")
		if v1 == toSearch {
			return v
		}
	}
	return nil
}

func (i *Inode) Delete() error {
	i.Name[0] = 0xE5
	return nil
}
