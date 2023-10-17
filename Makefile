VERSION ?= "$(shell git describe --tags --abbrev=0 | cut -c2-)"
COMMIT_HASH ?= "$(shell git describe --long --dirty --always --match "" || true)"

LDFLAGS ?= -s -w \
-X github.com/primevprotocol/mev-commit.version="$(VERSION)" \
-X github.com/primevprotocol/mev-commit.commitHash="$(COMMIT_HASH)"

.PHONY: build
build: export CGO_ENABLED=0
build: bin
	go build -ldflags '$(LDFLAGS)' -o bin/mev-commit ./cmd
	go build -ldflags '$(LDFLAGS)' -o bin/searcher-cli ./cmd/cli

bin:
	mkdir $@
