package impdos

import (
	"encoding/binary"
	"log"
	"testing"
)

func TestLoad(t *testing.T) {
	device := "/Users/jeromelesaux/Downloads/impdos_dump.img"
	imp, err := Read(device)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	ok, err := imp.Check()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if !ok {
		t.Fatalf("Expected ok and it is not.")
	}

	a, err := imp.ReadAutoExec()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	t.Logf("%s\n", a.String())
	if err := imp.ReadRootCatalogue(0); err != nil {
		t.Fatalf("%v\n", err)
	}
	log.Printf("%s\n", imp.Partitions[0].ListCatalogue())

	maxCluster, err := imp.Partitions[0].GetNextN(imp.Pointer)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	log.Printf("Next cluster is %d\n", maxCluster)
}

func TestLoadPartition1(t *testing.T) {
	device := "/Users/jeromelesaux/Downloads/impdos_master_dump.img"
	imp, err := Read(device)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	ok, err := imp.Check()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if !ok {
		t.Fatalf("Expected ok and it is not.")
	}

	a, err := imp.ReadAutoExec()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	t.Logf("%s\n", a.String())
	if err := imp.ReadCatalogues(); err != nil {
		t.Fatalf("%v\n", err)
	}
	log.Printf("%s\n", imp.Partitions[1].ListCatalogue())
	log.Printf("%s\n", imp.Partitions[2].ListCatalogue())
	maxCluster, err := imp.Partitions[0].GetNextN(imp.Pointer)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	log.Printf("Next cluster is %d\n", maxCluster)
}

// catalogue de Chany 0x2efa00 + 0x8000000

func TestReplaceName(t *testing.T) {
	v := ToImpdosName("/Users/jeromelesaux/Downloads/impdos_master_dump.BAS", false)
	log.Printf("[%s]\n", v)
	v = ToImpdosName("/Users/jeromelesaux/Downloads/GFX_KRIS", true)
	log.Printf("[%s]\n", v)

}

func TestCopyFileInDom(t *testing.T) {
	device := "/Users/jeromelesaux/Downloads/impdos_copy_dump.img"
	imp, err := Read(device)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	ok, err := imp.Check()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if !ok {
		t.Fatalf("Expected ok and it is not.")
	}
	if err := imp.ReadCatalogues(); err != nil {
		t.Fatal(err)
	}

	rootFolder := imp.Partitions[0].Inode
	if err := imp.Partitions[0].Save("/Users/jeromelesaux/Documents/Projets/go/src/github.com/jeromelesaux/impdos/ironman.scr", imp.Pointer, rootFolder); err != nil {
		t.Fatal(err)
	}
}

func TestCopyFileAndCreateFolderInDom(t *testing.T) {
	device := "/Users/jeromelesaux/Downloads/impdos_newfolder.img"
	imp, err := Read(device)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	ok, err := imp.Check()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if !ok {
		t.Fatalf("Expected ok and it is not.")
	}
	if err := imp.ReadCatalogues(); err != nil {
		t.Fatal(err)
	}

	rootFolder := imp.Partitions[0].Inode
	if err := imp.Partitions[0].Save("/Users/jeromelesaux/Documents/Projets/go/src/github.com/jeromelesaux/impdos/ironman.scr", imp.Pointer, rootFolder); err != nil {
		t.Fatal(err)
	}

	if _, err := imp.Partitions[0].NewFolder("TEST", imp.Pointer, imp.Partitions[0].Inode); err != nil {
		t.Fatal(err)
	}

	testInode := imp.Partitions[0].Inode.FindInode([]byte("TEST"))
	if err := imp.Partitions[0].Save("/Users/jeromelesaux/Documents/Projets/go/src/github.com/jeromelesaux/impdos/ironman.scr", imp.Pointer, testInode); err != nil {
		t.Fatal(err)
	}
}

func Test24bits(t *testing.T) {
	var size uint32 = 0x8000

	b := make([]byte, 4)
	b[1] = 0x80
	size2 := binary.LittleEndian.Uint32(b)
	if size2 != size {
		t.Fatal()
	}
	b2 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b2, size2)
	t.Log("")
}
