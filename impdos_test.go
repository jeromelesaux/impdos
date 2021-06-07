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
}

// catalogue de Chany 0x2efa00 + 0x8000000
