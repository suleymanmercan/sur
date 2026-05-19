.PHONY: build test lint run clean install uninstall purge

BINARY := sur
GO     := go
PREFIX ?= /usr/local
STATE_DIR ?= /var/lib/sur
LEGACY_TASK_DIR ?= /etc/sur

build:
	$(GO) build -trimpath -ldflags "-s -w" -o $(BINARY) .

test:
	$(GO) test ./... -count=1

lint:
	$(GO) vet ./...

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
