#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libusb.h"
//gcc dumper.c -o dumper `pkg-config --cflags --libs --static libusb-1.0`
//gcc dumper.c -o impdumper.exe -I"C:\msys64\clang64\include\libusb-1.0" -L"C:\msys64\clang64\lib" -l"usb-1.0" 

libusb_device *impdos_device = NULL;


static void usage() {
	printf("Dumper usage : \n");
	printf(">	dumper read|write impdos_image.ibc\n");
	printf(" for dumping data from usb device to image :\n");
	printf(">	dumper read my_impdos_image.ibc\n");
	printf(" for dumping data from image disk to usb device :\n");
	printf(">	dumper write my_impdos_image.ibc\n");
	exit(-1);
}

static void print_devs(libusb_device **devs)
{
	libusb_device *dev;
	int i = 0, j = 0;
	uint8_t path[8]; 


	while ((dev = devs[i++]) != NULL) {
		struct libusb_device_descriptor desc;
		int r = libusb_get_device_descriptor(dev, &desc);
		if (r < 0) {
			fprintf(stderr, "failed to get device descriptor");
			return;
		}
		if (desc.idVendor == 0x152d && desc.idProduct == 0x2338) {
			printf("Found ImpDOS device: ");
			impdos_device = dev;
		}
		printf("%04x:%04x (bus %d, device %d)",
			desc.idVendor, desc.idProduct,
			libusb_get_bus_number(dev), libusb_get_device_address(dev));
		
		r = libusb_get_port_numbers(dev, path, sizeof(path));
		if (r > 0) {
			printf(" path: %d", path[0]);
			for (j = 1; j < r; j++)
				printf(".%d", path[j]);
		}
		printf("\n");
	}
}


int setupDevice(libusb_device_handle *device) {
    printf("Setup usb device configuration\n");
	libusb_set_auto_detach_kernel_driver(device, 1);
	int status = libusb_claim_interface(device, 0);
	if (status != LIBUSB_SUCCESS) {
		libusb_close(device);
		printf("libusb_claim_interface failed: %s\n", libusb_error_name(status));
		return -1;
	}
	return 0;
}


void readData(libusb_device_handle *device, FILE *f) {
    printf("Read usb device\n");
	unsigned char buffer[512];
	int actual_length;
	while (libusb_interrupt_transfer(device, LIBUSB_ENDPOINT_IN, buffer, sizeof(buffer), &actual_length, 0)==0) {
		fwrite(buffer,sizeof(buffer),1,f);
	}
}


void writeData(libusb_device_handle *device, FILE *f) {
    printf("Write usb device\n");
	unsigned char buffer[512];
	int actual_length;
	while ( fread(buffer,sizeof(buffer),1,f) ) {
		libusb_interrupt_transfer(device, LIBUSB_ENDPOINT_OUT, buffer, sizeof(buffer), &actual_length, 0);		
	}
}

int main(int argc, char** argv)
{
	libusb_device **devs;
	int r;
	ssize_t cnt;
	FILE * imageFile; 
	//libusb_context *usbContext = NULL;
	void (*func_ptr)(libusb_device_handle *,FILE *) = NULL;

	if (argc != 3) {
		usage();
	}
	if (strcmp(argv[1],"read")) {
		imageFile = fopen(argv[2],"r"); 
		if (imageFile == NULL) {
			perror(argv[2]);
			exit(-1);
		}
		func_ptr = &readData;
	}
	if (strcmp(argv[1],"write")) {
		imageFile = fopen(argv[2],"w"); 
		if (imageFile == NULL) {
			perror(argv[2]);
			exit(-1);
		}
		func_ptr = &writeData;
	}
	
	r = libusb_init(NULL);
	if (r < 0)
		return r;
	//libusb_set_debug(usbContext, 3);
	printf("Scanning usb devices ...\n");
	cnt = libusb_get_device_list(NULL, &devs);
	if (cnt < 0){
		printf("No usb devices found\n");
		libusb_exit(NULL);
		return (int) cnt;
	}

	printf("Found [%d] usb devices\n",(int)cnt);

	print_devs(devs);

	if (impdos_device != NULL) {
		libusb_device_handle *device; 
		int status = libusb_open(impdos_device,&device);
		if (status==0) {
			int status = setupDevice(device);
			if (status==0) {
				// ok now I can write or read data from device.
				(*func_ptr)(device,imageFile);
			}
		} else {
            printf("Cannot open device error :%d\n",status);
        }	
	}
	libusb_free_device_list(devs, 1);

	libusb_exit(NULL);
	fclose(imageFile);
	return 0;
}

