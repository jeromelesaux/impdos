package usb

import (
	"bufio"
	"strings"
	"testing"
)

func TestMacosUsbDetection(t *testing.T) {
	devices, err := macosDetect()
	if err != nil {
		t.Fatalf("%v", err)
	}
	for _, v := range devices {
		t.Logf("Device found :%v\n", v)
	}
}

func TestDevicesDetect(t *testing.T) {
	devices, err := DevicesDetect()
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("Device found :%v\n", devices)
}

func TestLinuxUsbDetection(t *testing.T) {
	var devices []string
	out := `The backup GPT table is corrupt, but the primary appears OK, so that will be used.
Disk /dev/sdb: 2 TiB, 2199023185920 bytes, 4294967160 sectors
Disk model: Mass Storage
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
Disklabel type: gpt
Disk identifier: B9630062-166A-4D00-A9D2-04488AF53FD9
	
Device     Start        End    Sectors Size Type
/dev/sdb1     34 4294967126 4294967093   2T Microsoft basic data
The backup GPT table is corrupt, but the primary appears OK, so that will be used.
Disk /dev/sdc: 2 TiB, 2199023185920 bytes, 4294967160 sectors
Disk model: Mass Storage
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
Disklabel type: gpt
Disk identifier: DA9B68A3-758B-4F86-9E46-3113A81F107C

Device     Start        End    Sectors Size Type
/dev/sdc1     34 4294967126 4294967093   2T Microsoft basic data`
	scanner := bufio.NewScanner(strings.NewReader(out))
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
	t.Logf("Device found :%v\n", devices)

}
