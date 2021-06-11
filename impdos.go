package impdos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
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

func EmptyName() []byte {
	b := make([]byte, 11)
	for i := 0; i < 11; i++ {
		b[i] = 0x20
	}
	return b
}

func ToImpdosName(filePath string, isDirectory bool) []byte {
	name := EmptyName()
	filePath = strings.ToUpper(filePath)
	filePath = strings.ReplaceAll(filePath, "_", "")
	filePath = strings.ReplaceAll(filePath, " ", "")
	filePath = strings.ReplaceAll(filePath, "-", "")
	if isDirectory {
		v := path.Base(filePath)
		l := 11
		if len(v) < 11 {
			l = len(v)
		}
		copy(name, v[0:l])
	} else {
		v := path.Base(filePath)
		e := path.Ext(v)
		vv := strings.Replace(v, e, "", -1)
		l := 8
		if len(vv) < 8 {
			l = len(vv)
		}
		copy(name[8:], e[1:])
		copy(name[0:], vv[:l])
	}

	return name
}

type Partition struct {
	Inode           *Inode
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

func (imp *Impdos) ReadCatalogues() error {
	for i := 0; i < len(imp.Partitions); i++ {
		err := imp.ReadRootCatalogue(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (imp *Impdos) ReadRootCatalogue(partitionNumber int) error {
	return imp.Partitions[partitionNumber].ReadRootCatalogue(imp.Pointer)
}

func (p *Partition) PartitionOffset() int64 {
	return (0x8000000 * int64(p.PartitionNumber))
}

func (p *Partition) ReadRootCatalogue(f *os.File) error {

	var offset int64 = p.PartitionOffset() + 0x40200
	_, err := f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	inode := NewInode(p.PartitionOffset())
	inode.IsRoot = true
	if err := inode.ReadCatalogue(f); err != nil {
		return err
	}
	p.Inode = InitInode(p.PartitionOffset(), 2, 0, DirectoryType, []byte{})
	p.Inode.IsRoot = true
	p.Inode.Inodes = append(p.Inode.Inodes, inode.Inodes...)

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

func (p *Partition) ListCatalogue() string {
	var c string
	c += p.Inode.ListCatalogue("")
	/*for _, v := range p.Inode.Inodes {
		if v.IsDir() {
			c += fmt.Sprintf("[%.8s]\n", string(v.Name))
			c += v.ListCatalogue("\t")
		} else {
			c += fmt.Sprintf("%.8s %.4d Ko\n", string(v.Name), v.Size/1000)
		}
	}*/
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

func (p *Partition) SaveN(f *os.File, cluster uint16, size uint32) error {
	partitionOffset := p.PartitionOffset()
	sector1 := partitionOffset + 0x201

	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = f.Seek(sector1, io.SeekStart)
	if err != nil {
		return err
	}

	if err := binary.Write(f, binary.LittleEndian, &cluster); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, &size); err != nil {
		return err
	}
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func (p *Partition) SaveInode(fp *os.File, folder *Inode, newInode *Inode) error {
	offset, err := fp.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	catalogueOffset := folder.ClusterOffset()
	_, err = fp.Seek(int64(catalogueOffset), io.SeekStart)
	if err != nil {
		return err
	}
	// loop to find a new empty entry
	for {
		inode := NewInode(folder.PartitionOffset)
		if err := inode.Read(fp); err != nil {
			return err
		}
		if inode.IsEnd() {
			break
		}
	}

	// go to the start of the inode entry position in dom
	_, err = fp.Seek(-32, io.SeekCurrent)
	if err != nil {
		return err
	}
	if err := newInode.Save(fp); err != nil {
		return err
	}
	_, err = fp.Seek(int64(offset), io.SeekStart) // return to initial offset
	if err != nil {
		return err
	}
	return nil
}

func (p *Partition) GetNextN(f *os.File) (uint16, error) {
	n, err := p.GetLastN(f)
	if err != nil {
		return n, err
	}
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return n, err
	}
	partitionOffset := p.PartitionOffset()
	_, err = f.Seek(partitionOffset+0x203, io.SeekStart)
	if err != nil {
		return n, err
	}
	b := make([]byte, 4)
	if err := binary.Read(f, binary.LittleEndian, &b); err != nil {
		return n, err
	}
	size := binary.LittleEndian.Uint32(b)
	diff := size / 0x200
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return n, err
	}
	return n + uint16(diff) + 1, nil
}

func (p *Partition) GetLastN(f *os.File) (uint16, error) {
	partitionOffset := p.PartitionOffset()
	sector1 := partitionOffset + 0x201
	var cluster uint16
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return cluster, err
	}
	_, err = f.Seek(sector1, io.SeekStart)
	if err != nil {
		return cluster, err
	}
	if err := binary.Read(f, binary.LittleEndian, &cluster); err != nil {
		return cluster, err
	}
	_, err = f.Seek(offset, io.SeekStart)
	if err != nil {
		return cluster, err
	}
	if cluster == 0 {
		cluster = 2
	}
	return cluster, nil
}

func NewPartition(number int) *Partition {
	p := &Partition{
		PartitionNumber: number,
	}
	p.Inode = NewInode(p.PartitionOffset())
	return p
}

func Read(device string) (*Impdos, error) {
	var err error
	imp := NewImpdos()
	imp.Pointer, err = os.OpenFile(device, os.O_RDWR, os.ModePerm)
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

func (i *Inode) ClusterOffset() int {
	if i.IsRoot {
		return 0x40200 + int(i.PartitionOffset)
	} else {
		return (((int(i.Cluster)-2)*4)+0x221)*0x200 + int(i.PartitionOffset)
	}
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
	nextCatalogueOffset := i.ClusterOffset()

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

func (i *Inode) Put(f *os.File, data []byte) error {

	if i.IsDir() {
		return errors.New("inode is directory can not be get")
	}

	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	nextCatalogueOffset := i.ClusterOffset()

	/*	fmt.Printf("Name:%s Offset :%x next catalogue offset :%x\n",
		string(inode.Name),
		offset,
		nextCatalogueOffset)*/
	_, err = f.Seek(int64(nextCatalogueOffset), io.SeekStart)
	if err != nil {
		return err
	}

	read, err := f.Write(data)
	if err != nil {
		return err
	}
	if read != len(data) {
		return errors.New("read bytes differs from size inode")
	}

	_, err = f.Seek(int64(offset), io.SeekStart) // return to initial offset
	if err != nil {
		return err
	}
	return nil
}

type Inode struct {
	Name            []byte
	Type            byte   // 0 is a file #10 is a directory
	Unused          []byte // constant
	Cluster         uint16 // cluster number
	Size            uint32
	Inodes          []*Inode // files in the directory
	PartitionOffset int64
	IsRoot          bool
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

func isPrint(v []byte) bool {
	for _, c := range v {
		b := make([]byte, 1)
		b[0] = c
		r, _ := utf8.DecodeRune(b)
		if !strconv.IsPrint(r) {
			return false
		}
	}

	return true
}

func (i *Inode) ReadCatalogue(f *os.File) error {
	for {
		inode := NewInode(i.PartitionOffset)
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

func NewInode(partitionOffset int64) *Inode {
	return &Inode{
		Name:            make([]byte, 11),
		Unused:          make([]byte, 14),
		Inodes:          make([]*Inode, 0),
		PartitionOffset: partitionOffset,
	}
}

func InitInode(partitionOffset int64, cluster uint16, size uint32, inodeType byte, name []byte) *Inode {
	inode := NewInode(partitionOffset)
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

// secteur du catalgoue root toujours en secteur 201 soit offset 0x200*0x201 (512*513) 262656

func (i *Inode) findInode(name []byte) *Inode {
	toSearch := strings.Trim(string(name), " ")
	for _, v := range i.Inodes {
		v1 := strings.Trim(string(v.Name), " ")
		if v1 == toSearch {
			return v
		}
	}
	return nil
}

func (p *Partition) DeleteInode(inodeToDelete *Inode, folder *Inode, fp *os.File) error {
	inodeToDelete.Delete()
	offset, err := fp.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	catalogueOffset := folder.ClusterOffset()
	_, err = fp.Seek(int64(catalogueOffset), io.SeekStart)
	if err != nil {
		return err
	}

	// loop to find a new empty entry
	for {
		inode := NewInode(folder.PartitionOffset)
		if err := inode.Read(fp); err != nil {
			return err
		}
		if inode.Cluster == inodeToDelete.Cluster && inode.Size == inodeToDelete.Size {
			break
		}
	}

	// go to the start of the inode entry position in dom
	_, err = fp.Seek(-32, io.SeekCurrent)
	if err != nil {
		return err
	}
	// apply on dom the deleted inode
	if err := inodeToDelete.Save(fp); err != nil {
		return err
	}

	_, err = fp.Seek(int64(offset), io.SeekStart) // return to initial offset
	if err != nil {
		return err
	}

	return nil
}

func (i *Inode) Delete() error {
	i.Name[0] = 0xE5
	return nil
}

func (p *Partition) FormatCatalogue(fp *os.File, folder *Inode) error {

	offset := folder.ClusterOffset()
	orig, err := fp.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = fp.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return err
	}
	b := make([]byte, (4 * 0x200))
	for i := 0; i < (4 * 0x200); i++ {
		b[i] = 0xff
	}
	err = binary.Write(fp, binary.BigEndian, &b)
	if err != nil {
		return err
	}
	_, err = fp.Seek(orig, io.SeekStart)
	if err != nil {
		return err
	}
	return nil
}

func (p *Partition) NewFolder(folderName string, fp *os.File, folder *Inode) error {
	// transform file name
	impdosName := ToImpdosName(folderName, true)
	// get next cluster
	nextCluster, err := p.GetNextN(fp)
	if err != nil {
		return err
	}

	// add new inode
	newInode := InitInode(p.PartitionOffset(), nextCluster, 0, DirectoryType, impdosName)

	// insert new inode in catalogue
	folder.Inodes = append(folder.Inodes, newInode)
	if len(folder.Inodes) > 64 && p.PartitionNumber != 0 {
		return errors.New("catalogue exceed 64 entries")
	}

	if p.PartitionNumber == 0 && len(folder.Inodes) > 511 {
		return errors.New("catalogue exceed 511 entries")
	}
	// format track

	if err := p.FormatCatalogue(fp, newInode); err != nil {
		return err
	}

	// save on disk
	if err := p.SaveInode(fp, folder, newInode); err != nil {
		return err
	}
	// save last file cluster and size file in sector 1
	if err := p.SaveN(fp, nextCluster, 0); err != nil {
		return err
	}
	// get next cluster
	nextCluster, err = p.GetNextN(fp)
	if err != nil {
		return err
	}

	// now create the trash folder for the new folder
	// add Trash folder into newInode
	trashName := ToImpdosName("TRASH", true)
	originalTrash := folder.findInode([]byte("TRASH"))
	if originalTrash == nil {
		return errors.New("folder does not contain any trash folder")
	}
	trashInode := InitInode(p.PartitionOffset(), originalTrash.Cluster, 0, DirectoryType, trashName)

	// save on disk
	if err := p.SaveInode(fp, newInode, trashInode); err != nil {
		return err
	}
	// save last file cluster and size file in sector 1
	if err := p.SaveN(fp, nextCluster, 0); err != nil {
		return err
	}

	// get next cluster
	nextCluster, err = p.GetNextN(fp)
	if err != nil {
		return err
	}

	// now upper inode
	upperFolder := ToImpdosName("..", true)
	upperInode := InitInode(p.PartitionOffset(), folder.Cluster, 0, DirectoryType, upperFolder)

	// save on disk
	if err := p.SaveInode(fp, newInode, upperInode); err != nil {
		return err
	}
	// save last file cluster and size file in sector 1
	if err := p.SaveN(fp, nextCluster, 0); err != nil {
		return err
	}

	return nil
}

func (p *Partition) Save(filename string, fp *os.File, folder *Inode) error {
	// transform file name
	impdosName := ToImpdosName(filename, false)

	// read local file content
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	// get next cluster
	nextCluster, err := p.GetNextN(fp)
	if err != nil {
		return err
	}

	// add new inode
	newInode := InitInode(p.PartitionOffset(), nextCluster, uint32(len(b)), FileType, impdosName)

	// insert new inode in catalogue
	folder.Inodes = append(folder.Inodes, newInode)
	if len(folder.Inodes) > 64 && p.PartitionNumber != 0 {
		return errors.New("catalogue exceed 64 entries")
	}

	// save on disk
	if err := p.SaveInode(fp, folder, newInode); err != nil {
		return err
	}
	// copy file content in new sector
	if err := newInode.Put(fp, b); err != nil {
		return err
	}

	// save last file cluster and size file in sector 1
	if err := p.SaveN(fp, nextCluster, uint32(len(b))); err != nil {
		return err
	}

	return nil
}
