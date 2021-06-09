package impdos

import (
	"fmt"
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
	fmt.Printf("%s\n", imp.Partitions[0].ListCatalogue())

	maxCluster, err := imp.Partitions[0].GetNextN(imp.Pointer)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	fmt.Printf("Next cluster is %d\n", maxCluster)
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
	fmt.Printf("%s\n", imp.Partitions[1].ListCatalogue())
	fmt.Printf("%s\n", imp.Partitions[2].ListCatalogue())
	maxCluster, err := imp.Partitions[0].GetNextN(imp.Pointer)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	fmt.Printf("Next cluster is %d\n", maxCluster)
}

// catalogue de Chany 0x2efa00 + 0x8000000

func TestReplaceName(t *testing.T) {
	v := ToImpdosName("/Users/jeromelesaux/Downloads/impdos_master_dump.BAS", false)
	fmt.Printf("[%s]\n", v)
	v = ToImpdosName("/Users/jeromelesaux/Downloads/GFX_KRIS", true)
	fmt.Printf("[%s]\n", v)

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
