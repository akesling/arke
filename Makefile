PROJECT_DIR=arke
SOURCE_DIR=src/$(PROJECT_DIR)
BIN_DIR=bin
PKG_DIR=pkg
GO_PATH=$(shell pwd)
CC=go build
BUILD_DEPS=go get

all: daemon

% :: $(SOURCE_DIR)/% $(BIN_DIR) $(PKG_DIR)
	export GOPATH=$(GO_PATH)
	$(BUILD_DEPS) $(PROJECT_DIR)/$@
	$(CC) -o $(BIN_DIR)/$@ $(PROJECT_DIR)/$@

$(BIN_DIR) :
	mkdir -p $(BIN_DIR)

clean_bin :
	rm -f bin/*
	rmdir bin

$(PKG_DIR) :
	mkdir -p $(PKG_DIR)

clean_pkg :
	rm -rf pkg/*
	rmdir pkg

clean : clean_bin clean_pkg

fmt :
	gofmt -tabwidth=4 -tabs=false -w $(SOURCE_DIR)
