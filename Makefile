BINARY   := rubicon
VERSION  ?= 1.0.0
PKG      := github.com/gitstq/Rubicon/internal/cli
LDFLAGS  := -s -w -X $(PKG).Version=$(VERSION)
GO       ?= go

.PHONY: all build test vet fmt lint clean release demo help

all: fmt vet test build

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY) ./cmd/rubicon

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

lint: vet
	$(GO) run ./cmd/rubicon --dir .rubicon lint

clean:
	rm -rf dist

release:
	VERSION=$(VERSION) bash scripts/build.sh

demo:
	bash examples/demo.sh

help:
	@echo "Targets: all build test vet fmt lint clean release demo"
