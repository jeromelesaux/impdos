package impdos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrorBadEnd = errors.New("structure does not have ended tag #FF")
)

type DomType int

var (
	Dom512Mo                = 1 // 128000000 octets * 8 == 1024000000 byte *4
	Dom128Mo                = 2 // 128000000 octets * 8 == 1024000000 byte
	ParitionSize            = 0x7CE4800
	UnusedConstant0         = []byte{0x0, 0x0, 0x5d, 0x0, 0x21, 0x36, 0x0, 0x0, 0x0, 0x0, 0x5d, 0x0, 0x21, 0x36}
	DirectoryType      byte = 0x10
	FileType           byte = 0
	EndOfCatalogueType byte = 0xFF
	UpperDirectoryName      = []byte{0x2e, 0x2e, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20} // ..
	TrashDirectoryName      = []byte{0x54, 0x52, 0x41, 0x53, 0x48, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20} // TRASH
)

type Sector struct {
	Data    [0x200]byte
	Cluster uint32
}

func NewSector() *Sector {
	return &Sector{}
}

type Partition struct {
	Sector0         *Sector
	Sector1         *Sector
	Sectors         []*Sector
	Inodes          []*Inode
	PartitionNumber int
}

type Impdos struct {
	CheckTag   []byte
	Partitions []*Partition
	Pointer    *os.File
}

func NewImpdos() *Impdos {
	return &Impdos{
		CheckTag:   make([]byte, 6),
		Partitions: make([]*Partition, 0),
	}
}

func (imp *Impdos) ReadRootCatalogue() error {
	_, err := imp.Pointer.Seek(0x40200, io.SeekStart)
	if err != nil {
		return err
	}
	return imp.Partitions[0].ReadRootCatalogue(imp.Pointer)
}

func (p *Partition) ReadRootCatalogue(f *os.File) error {
	inode := NewInode()
	if err := inode.ReadCatalogue(f); err != nil {
		return err
	}
	p.Inodes = append(p.Inodes, inode.Inodes...)

	return nil
}
func (i *Inode) ListCatalogue(space string) string {
	var c string
	for _, v := range i.Inodes {

		if v.IsDir() {
			c += fmt.Sprintf("%s[%.8s]\n", space, string(v.Name))
			c += v.ListCatalogue(space + "\t")
		} else {
			c += fmt.Sprintf("%s%.8s %.4d Ko\n", space, string(v.Name), v.Size/1000)
		}
	}
	return c
}

func (p *Partition) ListCatalogue() string {
	var c string
	for _, v := range p.Inodes {
		if v.IsDir() {
			c += fmt.Sprintf("[%.8s]\n", string(v.Name))
			c += v.ListCatalogue("\t")
		} else {
			c += fmt.Sprintf("%.8s %.4d Ko\n", string(v.Name), v.Size/1000)
		}
	}
	return c
}

type AutoExec struct {
	Mode   byte
	Border byte
	Paper  byte
	Ink    byte
	End    byte
}

func (a *AutoExec) String() string {
	return fmt.Sprintf("Mode: %d, Border: %d, Paper: %d, Ink: %d",
		a.Mode,
		a.Border,
		a.Paper,
		a.Ink,
	)
}

func (imp *Impdos) Check() (bool, error) {
	_, err := imp.Pointer.Seek(0, io.SeekStart)
	if err != nil {
		return false, err
	}
	if err := binary.Read(imp.Pointer, binary.LittleEndian, imp.CheckTag); err != nil {
		return false, err
	}
	if string(imp.CheckTag) != "iMPdos" {
		return false, nil
	}
	return true, nil
}

func (imp *Impdos) ReadAutoExec() (*AutoExec, error) {
	a := &AutoExec{}
	_, err := imp.Pointer.Seek(0x400, io.SeekStart)
	if err != nil {
		return a, err
	}
	if err := binary.Read(imp.Pointer, binary.LittleEndian, a); err != nil {
		return a, err
	}
	if a.End != 0xff {
		return a, ErrorBadEnd
	}
	return a, nil
}

func NewPartition(number int) *Partition {
	return &Partition{
		Sector0:         NewSector(),
		Sector1:         NewSector(),
		Sectors:         make([]*Sector, 0),
		Inodes:          make([]*Inode, 0),
		PartitionNumber: number,
	}
}

func Read(device string) (*Impdos, error) {
	var err error
	imp := NewImpdos()
	imp.Pointer, err = os.Open(device)
	if err != nil {
		return imp, err
	}
	_, err = imp.Pointer.Seek(0, io.SeekStart)
	if err != nil {
		return imp, err
	}
	nbOctets, err := imp.Pointer.Seek(0, io.SeekEnd)
	if err != nil {
		return imp, err
	}
	fmt.Printf("[IMPDOS] Nb Octets read :%d\n", nbOctets)
	if nbOctets != int64(ParitionSize) {
		nbPartition := nbOctets / int64(ParitionSize)
		fmt.Printf("[IMPDOS] found %d partition\n", nbPartition)
		for i := 0; i < int(nbPartition); i++ {
			imp.Partitions = append(imp.Partitions, NewPartition(i))
		}
	}
	return imp, err
}

func ClusterOffset(n uint16) int {
	return (((int(n) - 2) * 4) + 0x221) * 0x200
}

func (i *Inode) Get(f *os.File) ([]byte, error) {

	b := make([]byte, i.Size)

	if i.IsDir() {
		return b, errors.New("inode is directory can not be get")
	}

	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return b, err
	}
	nextCatalogueOffset := ClusterOffset(i.Cluster)

	/*	fmt.Printf("Name:%s Offset :%x next catalogue offset :%x\n",
		string(inode.Name),
		offset,
		nextCatalogueOffset)*/
	_, err = f.Seek(int64(nextCatalogueOffset), io.SeekStart)
	if err != nil {
		return b, err
	}

	read, err := f.Read(b)
	if err != nil {
		return b, err
	}
	if read != len(b) {
		return b, errors.New("read bytes differs from size inode")
	}

	_, err = f.Seek(int64(offset), io.SeekStart) // return to initial offset
	if err != nil {
		return b, err
	}
	return b, nil
}

type Inode struct {
	Name    []byte
	Type    byte   // 0 is a file #10 is a directory
	Unused  []byte // constant
	Cluster uint16 // cluster number
	Size    uint32
	Inodes  []*Inode // files in the directory
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
		inode := NewInode()
		if err := inode.Read(f); err != nil {
			return err
		}
		if inode.IsEnd() {
			break
		}
		if inode.Type == DirectoryType && inode.Name[0] != '.' && inode.Name[1] != '.' {
			offset, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}
			nextCatalogueOffset := ClusterOffset(inode.Cluster)

			/*	fmt.Printf("Name:%s Offset :%x next catalogue offset :%x\n",
				string(inode.Name),
				offset,
				nextCatalogueOffset)*/
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
		i.Inodes = append(i.Inodes, inode)
	}
	return nil
}

func (i *Inode) IsEnd() bool {
	return i.Type == EndOfCatalogueType
}

func NewInode() *Inode {
	return &Inode{
		Name:   make([]byte, 11),
		Unused: make([]byte, 14),
		Inodes: make([]*Inode, 0),
	}
}

func (in *Inode) IsDir() bool {
	return in.Type == DirectoryType
}

// secteur du catalgoue root toujours en secteur 201 soit offset 0x200*0x201 (512*513) 262656
