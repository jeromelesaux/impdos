package impdos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func getLinkerPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(ex) + string(filepath.Separator) + "implink.exe", nil
}

func readDomWin(startAddress, size int64) ([]byte, error) {
	var b bytes.Buffer
	out := make([]byte, size)
	exePath, err := getLinkerPath()
	if err != nil {
		return b.Bytes(), err
	}

	cmd := exec.Command(exePath, "read",
		strconv.FormatInt(startAddress, 10),
		strconv.FormatInt(size, 10))
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	fmt.Printf("[WINDOWS-LINKER] %v\n", cmd.Args)
	err = cmd.Run()
	copy(out, b.Bytes())
	return out, err
}

func writeDomWin(startAddress, size int64, buf []byte) error {
	b := bytes.NewBuffer(buf)
	exePath, err := getLinkerPath()
	if err != nil {
		// dump to a file the buffer
		f, err0 := os.Create("dump_dom.bin")
		if err0 != nil {
			return err
		}
		_, err0 = f.Write(buf)
		if err0 != nil {
			return err
		}
		f.Close()
		return err
	}

	cmd := exec.Command(exePath, "write",
		strconv.FormatInt(startAddress, 10),
		strconv.FormatInt(size, 10))
	fmt.Printf("[WINDOWS-LINKER] %v\n", cmd.Args)
	cmd.Stdin = b
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

func inquiringDomWin() (domSize, blockSize int64, err error) {
	exePath, err := getLinkerPath()
	if err != nil {
		return
	}
	var b bytes.Buffer
	cmd := exec.Command(exePath, "inquiring")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = &b
	err = cmd.Run()
	fmt.Printf("[WINDOWS-LINKER] %s\n", b.String())
	out := strings.Split(b.String(), " ")
	if len(out) == 3 {
		if out[0] == "OK" {
			cleaned := strings.Replace(out[1], "\n", "", -1)
			domSize, err = strconv.ParseInt(cleaned, 10, 64)
			if err != nil {
				return
			}
			cleaned = strings.Replace(out[2], "\n", "", -1)
			blockSize, err = strconv.ParseInt(cleaned, 10, 64)
			if err != nil {
				return
			}
		}
	}
	return
}
