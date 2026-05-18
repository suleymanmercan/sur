.PHONY: build test lint run clean install

BINARY := sur
GO     := go

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
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)
	mkdir -p /etc/sur/tasks
	cp -r tasks/*.yaml /etc/sur/tasks/
