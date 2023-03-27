package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/gousb"
	"github.com/google/gousb/usbid"
)

func main() {
	ctx := gousb.NewContext()
	ctx.Debug(4)
	defer ctx.Close()
	/*// OpenDevices is used to find the devices to open.
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// The usbid package can be used to print out human readable information.
		fmt.Printf("%03d.%03d %s:%s %s\n", desc.Bus, desc.Address, desc.Vendor, desc.Product, usbid.Describe(desc))
		fmt.Printf("  Protocol: %s\n", usbid.Classify(desc))

		// The configurations can be examined from the DeviceDesc, though they can only
		// be set once the device is opened.  All configuration references must be closed,
		// to free up the memory in libusb.
		for _, cfg := range desc.Configs {
			// This loop just uses more of the built-in and usbid pretty printing to list
			// the USB devices.
			fmt.Printf("  %s:\n", cfg)
			for _, intf := range cfg.Interfaces {
				fmt.Printf("    --------------\n")
				for _, ifSetting := range intf.AltSettings {
					fmt.Printf("    %s\n", ifSetting)
					fmt.Printf("      %s\n", usbid.Classify(ifSetting))
					for _, end := range ifSetting.Endpoints {
						fmt.Printf("      %s\n", end)
					}
				}
			}
			fmt.Printf("    --------------\n")
		}

		// After inspecting the descriptor, return true or false depending on whether
		// the device is "interesting" or not.  Any descriptor for which true is returned
		// opens a Device which is retuned in a slice (and must be subsequently closed).
		return false
	})

	// All Devices returned from OpenDevices must be closed.
	defer func() {
		for _, d := range devs {
			d.Close()
		}
	}()

	// OpenDevices can occasionally fail, so be sure to check its return value.
	if err != nil {
		log.Fatalf("list: %s", err)
	}

	for _, dev := range devs {
		// Once the device has been selected from OpenDevices, it is opened
		// and can be interacted with.
		_ = dev
	}*/
	device, err := scanUsbDevices(ctx)
	if err != nil {
		fmt.Printf("[DUMPER] error while scanning device :%v\n", err)
		os.Exit(1)
	}
	if err := checkImpdosVolume(device); err != nil {
		fmt.Printf("[DUMPER] error while checking device :%v\n", err)
		os.Exit(1)
	}
	if device != nil {
		device.Close()
		_ = device
	}

}

type contextReader interface {
	ReadContext(context.Context, []byte) (int, error)
}

func checkImpdosVolume(dev *gousb.Device) error {

	err := dev.SetAutoDetach(true)
	if err != nil {
		return err
	}
	intf, _, err := dev.DefaultInterface()
	if err != nil {
		return err
	}
	ep, err := intf.InEndpoint(1)
	if err != nil {
		return err
	}
	var rdr contextReader = ep
	s, err := ep.NewStream(1024, 512)
	if err != nil {
		fmt.Printf("[DUMPER CHECK] ep.NewStream(): %v÷n", err)
		return err
	}
	defer s.Close()
	rdr = s
	opCtx := context.Background()
	var done func()
	opCtx, done = context.WithTimeout(opCtx, 10)
	defer done()
	buf := make([]byte, 1024)
	for i := 0; i < 8; i++ {
		num, err := rdr.ReadContext(opCtx, buf)
		if err != nil {
			fmt.Printf("[DUMPER CHECK] Reading from device failed: %v", err)
			return err
		}
		os.Stdout.Write(buf[:num])
	}
	return nil
}

func scanUsbDevices(ctx *gousb.Context) (*gousb.Device, error) {
	var impdosDevice *gousb.Device
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return true
	})
	if err != nil {
		return impdosDevice, err
	}
	for _, dev := range devs {
		if strings.Contains(usbid.Describe(dev.Desc), "JMicron Technology Corp") {
			fmt.Printf("[DUMPER SCAN] found %s\n", usbid.Describe(dev.Desc))
			impdosDevice = dev
		} else {
			dev.Close()
			_ = dev
		}
	}
	return impdosDevice, nil
}
