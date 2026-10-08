BINARY := breakup
GO     := go

.PHONY: build test vet fmt install clean

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

install:
	$(GO) install .

clean:
	rm -f $(BINARY) coverage.out
