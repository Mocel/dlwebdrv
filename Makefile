EXEC := dlwebdrv
VER := 0.0.1
REVISION := $(shell git rev-parse --short HEAD)
OBJS := $(EXEC)
BINDIR := bin
SRCS := $(shell find . -name '*.go')
PKG := bitbucket.org/iid-inc/dlwebdrv

GO          = go
GO_BUILD    = $(GO) build
GOIMPORTS   = goimports
GO_LIST     = $(GO) list
GOLINT      = golangci-lint
GO_TEST     = $(GO) test -v
GO_VET      = $(GO) vet
GO_LDFLAGS  = -ldflags="-s -w -X main.Version=$(VER) -X main.Revision=$(REVISION)"
GO_PKGROOT  = ./...
GO_PACKAGES = $(shell $(GO_LIST) $(GO_PKGROOT) | grep -v /vendor/)

all: $(OBJS)

$(EXEC): $(SRCS)
	$(GO_BUILD) -o $(BINDIR)/$(EXEC) $(GO_LDFLAGS) ./

$(EXEC).exe: $(SRCS)
	GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BINDIR)/$(EXEC).exe $(GO_LDFLAGS) ./

.PHONY: run clean build build-win lint vet fmt test

run:
	$(GO) run $(GO_PKGROOT)

clean:
	go clean
	rm -f $(BINDIR)/$(EXEC) $(BINDIR)/$(EXEC).exe

build: $(EXEC)

build-win: $(EXEC).exe

lint:
	$(GOLINT) run -v --disable-all -E errcheck -E gosimple -E govet -E ineffassign -E staticcheck -E typecheck -E unused ./...

vet:
	$(GO_VET)

fmt:
	$(GOIMPORTS) -w -local "$(PKG)" .

test:
	$(GO_TEST) -vet=off -v $(GO_PKGROOT)
