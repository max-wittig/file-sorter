GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=file-sorter
BINARY_LINUX=$(BINARY_NAME)
BINARY_MAC=$(BINARY_NAME)-darwin
BINARY_WINDOWS=$(BINARY_NAME).exe
ZIP_LINUX=file-sorter-amd64-linux.zip
ZIP_WINDOWS=file-sorter-amd64-win.zip
ZIP_MAC=file-sorter-amd64-darwin.zip

all: test build
build: 
		$(GOBUILD) -o build/$(BINARY_LINUX) -v
test: 
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -rf build/
run: build
		build/$(BINARY_LINUX) ~/Downloads
run_mod: build
		build/$(BINARY_LINUX) --criteria mod /home/max/DocumentsCommon/resumake.io
deps:
		dep ensure

build-linux:
		mkdir -p build/
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o build/$(BINARY_LINUX) -v
		cd build && zip -r $(ZIP_LINUX) $(BINARY_LINUX)
build-windows:
		mkdir -p build/
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o build/$(BINARY_WINDOWS) -v
		cd build && zip -r $(ZIP_WINDOWS) $(BINARY_WINDOWS)
build-mac:
		mkdir -p build/
		GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -o build/$(BINARY_MAC) -v
		cd build && zip -r $(ZIP_MAC) $(BINARY_MAC)

build-all: build-linux build-windows build-mac
