package impdos

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func getLinkerPath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(ex) + string(filepath.Separator) + "implinker.exe", nil
}

func readDomWin(startAddress, size int64) ([]byte, error) {
	var b bytes.Buffer
	exePath, err := getLinkerPath()
	if err != nil {
		return b.Bytes(), err
	}

	cmd := exec.Command(exePath, "read",
		strconv.FormatInt(startAddress, 10),
		strconv.FormatInt(size, 10))
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return b.Bytes(), err
}

func writeDomWin(startAddress, size int64, buf []byte) error {
	b := bytes.NewBuffer(buf)
	exePath, err := getLinkerPath()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, "write",
		strconv.FormatInt(startAddress, 10),
		strconv.FormatInt(size, 10))
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
	cmd := exec.Command(exePath, "inquiring")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	return
}
