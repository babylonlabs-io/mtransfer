BUILDDIR ?= $(CURDIR)/build
GO_BIN := ${GOPATH}/bin
VERSION := $(shell git describe --tags --always | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags := -X github.com/cosmos/cosmos-sdk/version.Name=mtransfer \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=mtransferd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)
build_tags := $(BUILD_TAGS)
build_args := $(BUILD_ARGS)

ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static" -v
endif

ifeq ($(VERBOSE),true)
	build_args += -v
endif

BUILD_TARGETS := build install
BUILD_FLAGS := --tags "$(build_tags)" --ldflags '$(ldflags)'

all: build install

build: clean_build $(BUILDDIR)/
	CGO_CFLAGS="-O -D__BLST_PORTABLE__" go build -mod=readonly $(BUILD_FLAGS) $(build_args) -o $(BUILDDIR) ./...

install: go.sum $(BUILDDIR)/
	CGO_CFLAGS="-O -D__BLST_PORTABLE__" go install -mod=readonly $(BUILD_FLAGS) $(build_args) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

clean_build:
	rm -rf $(BUILDDIR)