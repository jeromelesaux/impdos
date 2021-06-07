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
	if err := imp.ReadRootCatalogue(1); err != nil {
		t.Fatalf("%v\n", err)
	}
	fmt.Printf("%s\n", imp.Partitions[1].ListCatalogue())
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
