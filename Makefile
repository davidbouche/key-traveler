BINARY := ktraveler
PKG    := github.com/david/key-traveler

.PHONY: build build-strip install-usb test clean

build:
	go build -o $(BINARY) .

build-strip:
	go build -ldflags '-s -w' -trimpath -o $(BINARY) .

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
