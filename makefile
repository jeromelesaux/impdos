CC=go
RM=rm
MV=mv


SOURCEDIR=$(shell find . | grep main.go | sed s'/main\.go//')
SOURCES := $(shell find $(SOURCEDIR) -name 'main.go')



snapshot=$(shell date +%FT%T)
VERSION="1.0"

ifeq ($(suffix),rc)
	appversion=$(VERSION)$(snapshot)
else 
	appversion=$(VERSION)
endif 

.DEFAULT_GOAL:=build


build:
	@echo "Get the dependencies"
	go get fyne.io/fyne/v2/cmd/fyne
	go install fyne.io/fyne/v2/cmd/fyne
	@echo "Compilation for macos"
	gcc dumper/implink.c -o implink.exe `pkg-config --cflags --libs --static libusb-1.0`
	fyne package -os darwin -icon  $(SOURCEDIR)/Circle-icons-browser.png -name IMPBrowser -sourceDir $(SOURCEDIR)/
	zip -r IMPBrowser-$(appversion)-macos.zip IMPBrowser.app implink.exe
	@echo "Compilation for windows"
	x86_64-w64-mingw32-gcc dumper/implink.c -o implink.exe -I${HOME}/Downloads/libusb-1.0.24-src/out/include/libusb-1.0 -L${HOME}/Downloads/libusb-1.0.24-src/out/lib -lusb-1.0
	export GOOS=windows && export GOARCH=386 && export CGO_ENABLED=1 && export CC=i686-w64-mingw32-gcc && go build ${LDFLAGS} -o IMPBrowser.exe $(SOURCEDIR)/
	zip IMPBrowser-$(appversion)-windows.zip IMPBrowser.exe implink.exe

clean:
	@echo "Cleaning all *.zip archives."
	rm -f IMPBrowser*.zip
	@echo "Cleaning all binaries."
	rm -fr IMPBrowser*
