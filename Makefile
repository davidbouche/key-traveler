BINARY := ktraveler
PKG    := github.com/davidbouche/key-traveler
MAIN   := ./cmd/ktraveler

.PHONY: build build-strip install install-usb test vet clean

build:
	go build -o $(BINARY) $(MAIN)

build-strip:
	go build -ldflags '-s -w' -trimpath -o $(BINARY) $(MAIN)

# Install into $GOBIN (or $GOPATH/bin) at the pinned path used by `go install`.
install:
	go install $(MAIN)

install-usb:
	@test -n "$(USB)" || (echo "Usage: make install-usb USB=/media/<user>/<label>" && exit 1)
	cp $(BINARY) $(USB)/$(BINARY)
	chmod +x $(USB)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
