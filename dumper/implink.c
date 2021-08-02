#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stdarg.h>
#include <unistd.h>

#include "libusb.h"
//gcc dumper.c -o dumper `pkg-config --cflags --libs --static libusb-1.0`
//gcc dumper.c -o impdumper.exe -I"C:\msys64\clang64\include\libusb-1.0" -L"C:\msys64\clang64\lib" -l"usb-1.0" 
//gcc dumper.c -o impdumper.exe -I"C:\msys64\clang64\include\libusb-1.0" -L"C:\msys64\clang64\lib"  C:\msys64\clang64\lib\libusb-1.0.dll.a -static
//x86_64-w64-mingw32-gcc dumper.c -o impdumper.exe -I/Users/jeromelesaux/Downloads/libusb-1.0.24-src/out/include/libusb-1.0 -L/Users/jeromelesaux/Downloads/libusb-1.0.24-src/out/lib -lusb-1.0
libusb_device *impdos_device = NULL;
int BUFFERSIZE =  64;
int DEBUG = 0;
#define INQUIRY_LENGTH                0x24
#define READ_CAPACITY_LENGTH          0x08
// Mass Storage Requests values. See section 3 of the Bulk-Only Mass Storage Class specifications
#define BOMS_RESET                    0xFF
#define BOMS_GET_MAX_LUN              0xFE

#define RETRY_MAX                     5
#define REQUEST_SENSE_LENGTH          0x12

// Global variables
//static bool binary_dump = false;
//static const char* binary_name = NULL;

#define ERR_EXIT(errcode) do { perr("   %s\n", libusb_strerror((enum libusb_error)errcode)); return -1; } while (0)
#define CALL_CHECK(fcall) do { int _r=fcall; if (_r < 0) ERR_EXIT(_r); } while (0)
#define CALL_CHECK_CLOSE(fcall, hdl) do { int _r=fcall; if (_r < 0) { libusb_close(hdl); ERR_EXIT(_r); } } while (0)
#define be_to_int32(buf) (((buf)[0]<<24)|((buf)[1]<<16)|((buf)[2]<<8)|(buf)[3])
static void perr(char const *format, ...);

static const uint8_t cdb_length[256] = {
//	 0  1  2  3  4  5  6  7  8  9  A  B  C  D  E  F
	06,06,06,06,06,06,06,06,06,06,06,06,06,06,06,06,  //  0
	06,06,06,06,06,06,06,06,06,06,06,06,06,06,06,06,  //  1
	10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,  //  2
	10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,  //  3
	10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,  //  4
	10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,10,  //  5
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  6
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  7
	16,16,16,16,16,16,16,16,16,16,16,16,16,16,16,16,  //  8
	16,16,16,16,16,16,16,16,16,16,16,16,16,16,16,16,  //  9
	12,12,12,12,12,12,12,12,12,12,12,12,12,12,12,12,  //  A
	12,12,12,12,12,12,12,12,12,12,12,12,12,12,12,12,  //  B
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  C
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  D
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  E
	00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,00,  //  F
};

// Section 5.2: Command Status Wrapper (CSW)
struct command_status_wrapper {
	uint8_t dCSWSignature[4];
	uint32_t dCSWTag;
	uint32_t dCSWDataResidue;
	uint8_t bCSWStatus;
};

// Section 5.1: Command Block Wrapper (CBW)
struct command_block_wrapper {
	uint8_t dCBWSignature[4];
	uint32_t dCBWTag;
	uint32_t dCBWDataTransferLength;
	uint8_t bmCBWFlags;
	uint8_t bCBWLUN;
	uint8_t bCBWCBLength;
	uint8_t CBWCB[16];
};


static void usage() {
	fprintf(stderr,"Implink.exe usage : \n");
	fprintf(stderr,"#> Inquiring DOM :  > implink.exe inquiring\n");
	fprintf(stderr,"#> Reading part of the DOM : > implink.exe read start_address size > out.bin\n");
	fprintf(stderr,"#> Writing data in the DOM : > cat file.bin | implink.exe write start_address size\n");
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
			fprintf(stderr,"Found ImpDOS device: ");
			impdos_device = dev;
		}
		fprintf(stderr,"%04x:%04x (bus %d, device %d)",
			desc.idVendor, desc.idProduct,
			libusb_get_bus_number(dev), libusb_get_device_address(dev));
		
		r = libusb_get_port_numbers(dev, path, sizeof(path));
		if (r > 0) {
			fprintf(stderr," path: %d", path[0]);
			for (j = 1; j < r; j++) {
				fprintf(stderr,".%d", path[j]);
			}
		}
		for (int i1=0;i1<desc.bNumConfigurations;i1++) {
			struct libusb_config_descriptor *config;
			int ret = libusb_get_config_descriptor(dev, i1, &config);
			if ( LIBUSB_SUCCESS != ret) {
				break;
			}
			for (int j2=0; j2 <config->bNumInterfaces;j2++) {
				for (int j3=0; j3 <config->interface[j2].altsetting->bNumEndpoints;j3++) {
					fprintf(stderr,"\n\tendpoint [%d] address [0x%X]",j, config->interface[j2].altsetting->endpoint[j3].bEndpointAddress);				
				}
			}
			libusb_free_config_descriptor(config);
		}
		fprintf(stderr,"\n");
	}
}


int setupDevice(libusb_device_handle *device) {
    fprintf(stderr,"Setup usb device configuration\n");
	libusb_set_configuration(device,1);
	libusb_set_auto_detach_kernel_driver(device, 1);
	int status = libusb_claim_interface(device, 0);
	if (status != LIBUSB_SUCCESS) {
		libusb_close(device);
		if (DEBUG==1) {
			fprintf(stderr,"libusb_claim_interface failed: %s\n", libusb_error_name(status));
		}
		return -1;
	}
	return 0;
}


static int send_mass_storage_command(libusb_device_handle *handle, uint8_t endpoint, uint8_t lun,
	uint8_t *cdb, uint8_t direction, int data_length, uint32_t *ret_tag)
{
	static uint32_t tag = 1;
	uint8_t cdb_len;
	int i, r, size;
	struct command_block_wrapper cbw;

	if (cdb == NULL) {
		return -1;
	}

	if (endpoint & LIBUSB_ENDPOINT_IN) {
		perr("send_mass_storage_command: cannot send command on IN endpoint\n");
		return -1;
	}

	cdb_len = cdb_length[cdb[0]];
	if ((cdb_len == 0) || (cdb_len > sizeof(cbw.CBWCB))) {
		perr("send_mass_storage_command: don't know how to handle this command (%02X, length %d)\n",
			cdb[0], cdb_len);
		return -1;
	}

	memset(&cbw, 0, sizeof(cbw));
	cbw.dCBWSignature[0] = 'U';
	cbw.dCBWSignature[1] = 'S';
	cbw.dCBWSignature[2] = 'B';
	cbw.dCBWSignature[3] = 'C';
	*ret_tag = tag;
	cbw.dCBWTag = tag++;
	cbw.dCBWDataTransferLength = data_length;
	cbw.bmCBWFlags = direction;
	cbw.bCBWLUN = lun;
	// Subclass is 1 or 6 => cdb_len
	cbw.bCBWCBLength = cdb_len;
	memcpy(cbw.CBWCB, cdb, cdb_len);

	i = 0;
	do {
		// The transfer length must always be exactly 31 bytes.
		r = libusb_bulk_transfer(handle, endpoint, (unsigned char*)&cbw, 31, &size, 1000);
		if (r == LIBUSB_ERROR_PIPE) {
			libusb_clear_halt(handle, endpoint);
		}
		i++;
	} while ((r == LIBUSB_ERROR_PIPE) && (i<RETRY_MAX));
	if (r != LIBUSB_SUCCESS) {
		perr("   send_mass_storage_command: %s\n", libusb_strerror((enum libusb_error)r));
		return -1;
	}

	if (DEBUG==1) {
		fprintf(stderr,"   sent %d CDB bytes\n", cdb_len);
	}
	return 0;
}
static uint8_t* u32_to_u8(const uint32_t u32, uint8_t* u8) {
  // To extract each byte, we can mask them using bitwise AND (&)
  // then shift them right to the first byte.
  u8[0] = (u32 & 0xff000000) >> 24;
  u8[1] = (u32 & 0x00ff0000) >> 16;
  u8[2] = (u32 & 0x0000ff00) >> 8;
  u8[3] = u32 & 0x000000ff;
  return u8;
}

static void display_buffer_hex(unsigned char *buffer, unsigned size)
{
	unsigned i, j, k;

	for (i=0; i<size; i+=16) {
		fprintf(stderr,"\n  %08x  ", i);
		for(j=0,k=0; k<16; j++,k++) {
			if (i+j < size) {
				fprintf(stderr,"%02x", buffer[i+j]);
			} else {
				fprintf(stderr,"  ");
			}
			fprintf(stderr," ");
		}
		fprintf(stderr," ");
		for(j=0,k=0; k<16; j++,k++) {
			if (i+j < size) {
				if ((buffer[i+j] < 32) || (buffer[i+j] > 126)) {
					fprintf(stderr,".");
				} else {
					fprintf(stderr,"%c", buffer[i+j]);
				}
			}
		}
	}
	fprintf(stderr,"\n" );
}

static void perr(char const *format, ...)
{
	va_list args;

	va_start (args, format);
	fprintf(stderr, format, args);
	va_end(args);
}


static int get_mass_storage_status(libusb_device_handle *handle, uint8_t endpoint, uint32_t expected_tag)
{
	int i, r, size;
	struct command_status_wrapper csw;

	// The device is allowed to STALL this transfer. If it does, you have to
	// clear the stall and try again.
	i = 0;
	do {
		r = libusb_bulk_transfer(handle, endpoint, (unsigned char*)&csw, 13, &size, 1000);
		if (r == LIBUSB_ERROR_PIPE) {
			libusb_clear_halt(handle, endpoint);
		}
		i++;
	} while ((r == LIBUSB_ERROR_PIPE) && (i<RETRY_MAX));
	if (r != LIBUSB_SUCCESS) {
		perr("   get_mass_storage_status: %s\n", libusb_strerror((enum libusb_error)r));
		return -1;
	}
	if (size != 13) {
		perr("   get_mass_storage_status: received %d bytes (expected 13)\n", size);
		return -1;
	}
	if (csw.dCSWTag != expected_tag) {
		perr("   get_mass_storage_status: mismatched tags (expected %08X, received %08X)\n",
			expected_tag, csw.dCSWTag);
		return -1;
	}
	// For this test, we ignore the dCSWSignature check for validity...
	if (DEBUG==1) {
		fprintf(stderr,"   Mass Storage Status: %02X (%s)\n", csw.bCSWStatus, csw.bCSWStatus?"FAILED":"Success");
	}
	if (csw.dCSWTag != expected_tag)
		return -1;
	if (csw.bCSWStatus) {
		// REQUEST SENSE is appropriate only if bCSWStatus is 1, meaning that the
		// command failed somehow.  Larger values (2 in particular) mean that
		// the command couldn't be understood.
		if (csw.bCSWStatus == 1)
			return -2;	// request Get Sense
		else
			return -1;
	}

	// In theory we also should check dCSWDataResidue.  But lots of devices
	// set it wrongly.
	return 0;
}


static void get_sense(libusb_device_handle *handle, uint8_t endpoint_in, uint8_t endpoint_out)
{
	uint8_t cdb[16];	// SCSI Command Descriptor Block
	uint8_t sense[18];
	uint32_t expected_tag;
	int size;
	int rc;

	// Request Sense
	if (DEBUG==1) {
		fprintf(stderr,"Request Sense:\n");
	}
	memset(sense, 0, sizeof(sense));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x03;	// Request Sense
	cdb[4] = REQUEST_SENSE_LENGTH;

	send_mass_storage_command(handle, endpoint_out, 0, cdb, LIBUSB_ENDPOINT_IN, REQUEST_SENSE_LENGTH, &expected_tag);
	rc = libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&sense, REQUEST_SENSE_LENGTH, &size, 1000);
	if (rc < 0)
	{
		fprintf(stderr,"libusb_bulk_transfer failed: %s\n", libusb_error_name(rc));
		return;
	}
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	if ((sense[0] != 0x70) && (sense[0] != 0x71)) {
		perr("   ERROR No sense data\n");
	} else {
		perr("   ERROR Sense: %02X %02X %02X\n", sense[2]&0x0F, sense[12], sense[13]);
	}
	// Strictly speaking, the get_mass_storage_status() call should come
	// before these perr() lines.  If the status is nonzero then we must
	// assume there's no data in the buffer.  For xusb it doesn't matter.
	get_mass_storage_status(handle, endpoint_in, expected_tag);
}



// Mass Storage device to test bulk transfers (non destructive test)
static int read_mass_storage(libusb_device_handle *handle, uint8_t endpoint_in, uint8_t endpoint_out, FILE *f, int start_address, int size_expected)
{
	int r, size;
	uint8_t lun;
	uint32_t expected_tag;
	uint32_t i, max_lba, block_size;
	double device_size;
	uint8_t cdb[16];	// SCSI Command Descriptor Block
	uint8_t buffer[64];
	char vid[9], pid[9], rev[5];
	unsigned char *data;
	if (DEBUG==1) {
		fprintf(stderr,"Reading Max LUN:\n");
	}
	r = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE,
		BOMS_GET_MAX_LUN, 0, 0, &lun, 1, 1000);
	// Some devices send a STALL instead of the actual value.
	// In such cases we should set lun to 0.
	if (r == 0) {
		lun = 0;
	} else if (r < 0) {
		perr("   Failed: %s", libusb_strerror((enum libusb_error)r));
	}
	if (DEBUG==1) {
		fprintf(stderr,"   Max LUN = %d\n", lun);
	}
	// Send Inquiry
	if (DEBUG==1) {
		fprintf(stderr,"Sending Inquiry:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x12;	// Inquiry
	cdb[4] = INQUIRY_LENGTH;

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, INQUIRY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, INQUIRY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	// The following strings are not zero terminated
	for (i=0; i<8; i++) {
		vid[i] = buffer[8+i];
		pid[i] = buffer[16+i];
		rev[i/2] = buffer[32+i/2];	// instead of another loop
	}
	vid[8] = 0;
	pid[8] = 0;
	rev[4] = 0;
	if (DEBUG==1) {
		fprintf(stderr,"   VID:PID:REV \"%8s\":\"%8s\":\"%4s\"\n", vid, pid, rev);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}

	// Read capacity
	if (DEBUG==1) {
		fprintf(stderr,"Reading Capacity:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x25;	// Read Capacity

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, READ_CAPACITY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, READ_CAPACITY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	max_lba = be_to_int32(&buffer[0]);
	block_size = be_to_int32(&buffer[4]);
	device_size = ((double)(max_lba+1))*block_size/(1024*1024*1024);
	if (DEBUG==1) {
		fprintf(stderr,"   Max LBA: %08X, Block Size: %08X (%.2f GB)\n", max_lba, block_size, device_size);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}

	
	// Send Read
	if (DEBUG==1) {
		fprintf(stderr,"Attempting to read %u bytes:\n", block_size);
	}
	// coverity[tainted_data]
	data = (unsigned char*) calloc(1, block_size);
	if (data == NULL) {
		perr("   unable to allocate data buffer\n");
		return -1;
	}
	bool start_zero = true;
	uint32_t start_offset = 0;
	if (size_expected%block_size != 0) { // commence à un multiple de block_size ?
		start_zero = false;
		if (block_size > size_expected) {
			start_offset = start_address;
		} else {
			start_offset = size_expected % block_size;
		}
	}
    uint32_t start_block = (start_address / block_size);
	uint32_t nb_iter = (size_expected / block_size)+1 + start_block;
	uint32_t size_copied = 0;
	uint8_t *block_number;
	block_number = calloc(4,sizeof(uint8_t));
	if (DEBUG==1) {
		fprintf(stderr,"   NB iterations :%d, device size:%f, block_size:%08X\n",nb_iter,device_size,block_size);
	}
	for (i=start_block; i < nb_iter ; i++){ 
		memset(block_number,0,sizeof(block_number));
		block_number = u32_to_u8(i,block_number);

		memset(cdb, 0, sizeof(cdb));
		cdb[0] = 0x28;	// Read(10)
		cdb[2] = block_number[0]; // block number
		cdb[3] = block_number[1]; // 
		cdb[4] = block_number[2]; // 
		cdb[5] = block_number[3]; // 
		cdb[8] = 0x1;	// number of block to read 

		send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, block_size, &expected_tag);
		libusb_bulk_transfer(handle, endpoint_in, data, block_size, &size, 1000);
		usleep(200);
		if (DEBUG==1) {
			fprintf(stderr,"   READ: received %d bytes from block :%.2X,:%.2X,:%.2X,:%.2X\n", size, cdb[2], cdb[3], cdb[4], cdb[5]);
		}
		if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
			get_sense(handle, endpoint_in, endpoint_out);
		} else {
			if (DEBUG==1) {
				display_buffer_hex(data, size);
			}
			if (!start_zero) { // si pas multiple de block_size
				if (size_expected <block_size) { // si taille demandée < block_size
					if (fwrite(data[start_offset], 1, (size_t)size_expected, f) != (unsigned int)size_expected) {
						perr("   unable to write binary data\n");
					}
					size_copied += size_expected;
				} else {
					uint32_t s = size - start_offset;
					if (fwrite(data[start_offset], 1, (size_t)s, f) != (unsigned int)s) {
						perr("   unable to write binary data\n");
					}
				}
				start_zero = true;
			} else {
				if ((size_expected - size_copied) < block_size) {
					uint32_t s = size_expected - size_copied;
					if (fwrite(data, 1, (size_t)s, f) != (unsigned int)s) {
						perr("   unable to write binary data\n");
					}
					size_copied += s;
				} else {
					if (fwrite(data, 1, (size_t)size, f) != (unsigned int)size) {
						perr("   unable to write binary data\n");
					}
					size_copied += size;
				}
			}
			fflush(f);
		}
		memset(data, 0, sizeof(data));
	}
	
	free(block_number);
	free(data);
	return 0;
}


// Mass Storage device to test bulk transfers (non destructive test)
static int inquiring_mass_storage(libusb_device_handle *handle, uint8_t endpoint_in, uint8_t endpoint_out)
{
	int r, size;
	uint8_t lun;
	uint32_t expected_tag;
	uint32_t i, max_lba, block_size;
	uint64_t device_size;
	uint8_t cdb[16];	// SCSI Command Descriptor Block
	uint8_t buffer[64];
	char vid[9], pid[9], rev[5];
	unsigned char *data;
    fprintf(stderr,"Reading Max LUN:\n");
	r = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE,
		BOMS_GET_MAX_LUN, 0, 0, &lun, 1, 1000);
	// Some devices send a STALL instead of the actual value.
	// In such cases we should set lun to 0.
	if (r == 0) {
		lun = 0;
	} else if (r < 0) {
		perr("   Failed: %s", libusb_strerror((enum libusb_error)r));
	}
	if (DEBUG==1) {
		fprintf(stderr,"   Max LUN = %d\n", lun);
	}

	// Send Inquiry
	if (DEBUG==1) {
		fprintf(stderr,"Sending Inquiry:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x12;	// Inquiry
	cdb[4] = INQUIRY_LENGTH;

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, INQUIRY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, INQUIRY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	// The following strings are not zero terminated
	for (i=0; i<8; i++) {
		vid[i] = buffer[8+i];
		pid[i] = buffer[16+i];
		rev[i/2] = buffer[32+i/2];	// instead of another loop
	}
	vid[8] = 0;
	pid[8] = 0;
	rev[4] = 0;
	if (DEBUG==1) {
		fprintf(stderr,"   VID:PID:REV \"%8s\":\"%8s\":\"%4s\"\n", vid, pid, rev);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}

	// Read capacity
	if (DEBUG==1) {
		fprintf(stderr,"Reading Capacity:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x25;	// Read Capacity

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, READ_CAPACITY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, READ_CAPACITY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	max_lba = be_to_int32(&buffer[0]);
	block_size = be_to_int32(&buffer[4]);
	device_size = (max_lba+1)*block_size;
	if (DEBUG==1) {
		fprintf(stderr,"   Max LBA: %08X, Block Size: %08X (%.2f GB)\n", max_lba, block_size, device_size);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}
    fprintf(stdout,"OK %ld %d\n",device_size,block_size);
    return 0;
}


// Mass Storage device to test bulk transfers (non destructive test)
static int write_mass_storage(libusb_device_handle *handle, uint8_t endpoint_in, uint8_t endpoint_out, FILE *f, int start_address, int size_expected)
{
	int r, size;
	uint8_t lun;
	uint32_t expected_tag;
	uint32_t i, max_lba, block_size;
	double device_size;
	uint8_t cdb[16];	// SCSI Command Descriptor Block
	uint8_t buffer[64];
	char vid[9], pid[9], rev[5];
	unsigned char *data;

	fprintf(stderr,"Reading Max LUN:\n");
	r = libusb_control_transfer(handle, LIBUSB_ENDPOINT_IN|LIBUSB_REQUEST_TYPE_CLASS|LIBUSB_RECIPIENT_INTERFACE,
		BOMS_GET_MAX_LUN, 0, 0, &lun, 1, 1000);
	// Some devices send a STALL instead of the actual value.
	// In such cases we should set lun to 0.
	if (r == 0) {
		lun = 0;
	} else if (r < 0) {
		perr("   Failed: %s", libusb_strerror((enum libusb_error)r));
	}
	if (DEBUG==1) {
		fprintf(stderr,"   Max LUN = %d\n", lun);
	}

	// Send Inquiry
	if (DEBUG==1) {
		fprintf(stderr,"Sending Inquiry:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x12;	// Inquiry
	cdb[4] = INQUIRY_LENGTH;

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, INQUIRY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, INQUIRY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	// The following strings are not zero terminated
	for (i=0; i<8; i++) {
		vid[i] = buffer[8+i];
		pid[i] = buffer[16+i];
		rev[i/2] = buffer[32+i/2];	// instead of another loop
	}
	vid[8] = 0;
	pid[8] = 0;
	rev[4] = 0;
	if (DEBUG==1) {
		fprintf(stderr,"   VID:PID:REV \"%8s\":\"%8s\":\"%4s\"\n", vid, pid, rev);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}

	// Read capacity
	if (DEBUG==1) {
		fprintf(stderr,"Reading Capacity:\n");
	}
	memset(buffer, 0, sizeof(buffer));
	memset(cdb, 0, sizeof(cdb));
	cdb[0] = 0x25;	// Read Capacity

	send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, READ_CAPACITY_LENGTH, &expected_tag);
	CALL_CHECK(libusb_bulk_transfer(handle, endpoint_in, (unsigned char*)&buffer, READ_CAPACITY_LENGTH, &size, 1000));
	if (DEBUG==1) {
		fprintf(stderr,"   received %d bytes\n", size);
	}
	max_lba = be_to_int32(&buffer[0]);
	block_size = be_to_int32(&buffer[4]);
	device_size = ((double)(max_lba+1))*block_size/(1024*1024*1024);
	if (DEBUG==1) {
		fprintf(stderr,"   Max LBA: %08X, Block Size: %08X (%.2f GB)\n", max_lba, block_size, device_size);
	}
	if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
		get_sense(handle, endpoint_in, endpoint_out);
	}

	
	// Send Read
	if (DEBUG==1) {
		fprintf(stderr,"Attempt	ing to read %u bytes:\n", block_size);
	}
	// coverity[tainted_data]
	data = (unsigned char*) calloc(1, block_size);
	if (data == NULL) {
		perr("   unable to allocate data buffer\n");
		return -1;
	}
	
	uint32_t nb_iter = (size_expected / block_size)+1;
    uint32_t start_iter = (start_address / block_size);
	uint8_t *block_number;
	block_number = calloc(4,sizeof(uint8_t));
	if (DEBUG==1) {
		fprintf(stderr,"   NB iterations :%d, device size:%f, block_size:%08X\n",nb_iter,device_size,block_size);
	}
	for (i=start_iter; i < nb_iter ; i++){ 
		memset(block_number,0,sizeof(block_number));
		block_number = u32_to_u8(i,block_number);

		memset(cdb, 0, sizeof(cdb));
		cdb[0] = 0x2A;	// Read(10)
		cdb[2] = block_number[0]; // block number
		cdb[3] = block_number[1]; // 
		cdb[4] = block_number[2]; // 
		cdb[5] = block_number[3]; // 
		cdb[8] = 0x1;	// number of block to read 

		send_mass_storage_command(handle, endpoint_out, lun, cdb, LIBUSB_ENDPOINT_IN, block_size, &expected_tag);
		libusb_bulk_transfer(handle, endpoint_in, data, block_size, &size, 1000);
		usleep(200);
		if (DEBUG==1) {
			fprintf(stderr,"   READ: received %d bytes from block :%.2X,:%.2X,:%.2X,:%.2X\n", size, cdb[2], cdb[3], cdb[4], cdb[5]);
		}
		if (get_mass_storage_status(handle, endpoint_in, expected_tag) == -2) {
			get_sense(handle, endpoint_in, endpoint_out);
		} else {
			display_buffer_hex(data, size);
			if (fwrite(data, 1, (size_t)size, f) != (unsigned int)size) {
				perr("   unable to write binary data\n");
			}
			fflush(f);
		}
		memset(data, 0, sizeof(data));
	}
	
	free(block_number);
	free(data);
	return 0;
}



int main(int argc, char** argv)
{
	
	libusb_device **devs;
	int r;
	ssize_t cnt;
	FILE * output; 
	bool read = false;
    bool write = false;
    bool inquiring = false;
    int start_address=0;
    int size_expected=0;

	void (*func_ptr)(libusb_device_handle *,FILE *) = NULL;
    /*
        handle : 
        inquiring -> input -> inquiring -> ouput -> (OK or ERROR) size message
        read DOM -> input -> read start_address size -> output -> stdout
        write DOM -> input -> write start_address size -> output -> stdin
    */
   	fprintf(stderr,"Number of arguments %d\n",argc);
    if (argc == 2 || argc == 3 ) { // inquiring DOM
		if (strcmp(argv[1],"inquiring")==0) {
			output = stdout;
			inquiring = true;
		}
		if (argc == 3) {
			if (strcmp(argv[2],"debug")==0) {
				DEBUG = 1;
			}
		}
    } else {
        if (argc >= 4) { // read or write to DOM
            if (strcmp(argv[1],"read")==0) {
                fprintf(stderr,"Will read the DOM to the image file %s\n",argv[2]);
                output = stdout; 
				read = true;
            } else {
                if (strcmp(argv[1],"write")==0) {
                    fprintf(stderr,"Will write the image %s in to the DOM\n",argv[2]);
                    output = stdin;
					write = true;
                }
            }
            start_address = atoi(argv[2]);
            size_expected = atoi(argv[3]);

            if (argc == 5) {
                if (strcmp(argv[3],"debug")==0) {
                    DEBUG = 1;
                }
            }
        } else {
            usage();
        }
    }


	
	
	r = libusb_init(NULL);
	if (r < 0)
		return r;
	
	fprintf(stderr,"Scanning usb devices ...\n");
	cnt = libusb_get_device_list(NULL, &devs);
	if (cnt < 0){
		fprintf(stderr,"No usb devices found\n");
		libusb_exit(NULL);
		return (int) cnt;
	}

	fprintf(stderr,"Found [%d] usb devices\n",(int)cnt);

	print_devs(devs);

	if (impdos_device != NULL) {
		libusb_device_handle *device; 
		int status = libusb_open(impdos_device,&device);
		if (status==0) {
			int status = setupDevice(device);
			if (status==0) {		
				// ok now I can write or read data from device.

				if (read) {
					fprintf(stderr,"Get the DOM content.\n");
					read_mass_storage(device,0x81,0x02,output, start_address, size_expected);
				} else {
                    if (write) {
                        fprintf(stderr,"Burn the DOM.\n");
                        write_mass_storage(device,0x81,0x02,output, start_address, size_expected);
                    } else {
                        if (inquiring) {
							fprintf(stderr,"Inquiring the DOM.\n");
                            inquiring_mass_storage(device,0x81,0x02);
                        }
                    }
				}
				fprintf(stderr,"Process ended.\n");
			} else {
				fprintf(stderr,"Error while setting up the configuration. %s\n",libusb_error_name( status ));
			}
		} else {
            fprintf(stderr,"Cannot open device error :%s\n",libusb_error_name( status ));
        }	
	}
	libusb_free_device_list(devs, 1);
	libusb_exit(NULL);
	
	return 0;
}

