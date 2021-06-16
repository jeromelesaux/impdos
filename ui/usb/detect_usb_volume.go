package usb

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

/*
diskutil list external
/dev/disk2 (external, physical):
   #:                       TYPE NAME                    SIZE       IDENTIFIER
   0:                                                   *523.8 MB   disk2

   for macos /dev/disk2


   windows diskpart list disk
   get-psdrive -psprovider filesystem?
*/

func macosDetect() (devices []string, err error) {
	cmd := exec.Command("diskutil", "list", "external")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(strings.NewReader(out.String()))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		v := strings.Split(scanner.Text(), " ")
		if v[0] != "" {
			devices = append(devices, v[0])
		}
	}

	return
}

func linuxDetect() (devices []string, err error) {
	cmd := exec.Command("fdisk", "-l")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(strings.NewReader(out.String()))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		t := scanner.Text()
		var l int = 4
		if len(t) < 4 {
			l = len(t)
		}
		tag := t[0:l]
		if tag == "/dev" {
			v := strings.Split(t, " ")
			if v[0] != "" {
				devices = append(devices, v[0])
			}
		}
	}

	return
}

func DevicesDetect() (devices []string, err error) {
	os := runtime.GOOS
	switch os {
	case "darwin":
		return macosDetect()
	case "linux":
		return linuxDetect()
	default:
		err = errors.New("your OS is not supported for now")
	}

	return
}
