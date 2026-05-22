.PHONY: build test lint run clean install uninstall purge

BINARY := sur
GO     := go
PREFIX ?= /usr/local
STATE_DIR ?= /var/lib/sur
LEGACY_TASK_DIR ?= /etc/sur
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
LDFLAGS := -s -w -X github.com/suleymanmercan/sur/cmd.Version=$(VERSION)

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	$(GO) test ./... -count=1

lint:
	$(GO) vet ./...
lint2:
	golangci-lint fmt --diff ./...
	golangci-lint run
security:
	govulncheck ./...

run-check: build
	./$(BINARY) check

clean:
	rm -f $(BINARY)

install: build
	mkdir -p $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

purge: uninstall
	rm -rf $(LEGACY_TASK_DIR)
	rm -rf $(STATE_DIR)
