GO := $(shell command -v go 2>/dev/null || printf '%s' ../.tools/go/bin/go)
export GOCACHE ?= /tmp/mor10z-go-cache
PREFIX ?= /usr/local

.PHONY: build test check run clean install dist
build:
	$(GO) build -buildvcs=false -trimpath -o bin/mor10z ./cmd/mor10z
test:
	$(GO) test -buildvcs=false ./...
check:
	$(GO) vet -buildvcs=false ./...
	MOR10Z_AUDIO_TEST=1 $(GO) test -buildvcs=false -race ./...
run: build
	./bin/mor10z
clean:
	rm -rf bin
install:
	install -Dm755 bin/mor10z $(DESTDIR)$(PREFIX)/bin/mor10z
	install -Dm644 packaging/mor10z.desktop $(DESTDIR)$(PREFIX)/share/applications/mor10z.desktop
	install -Dm644 README.md $(DESTDIR)$(PREFIX)/share/doc/mor10z/README.md
dist:
	GO="$(GO)" bash packaging/dist.sh
