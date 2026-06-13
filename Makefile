.PHONY: all build test vet clean install bump-patch bump-minor bump-major release release-dry-run version help

BINARY   := traverse
VERSION  := $(shell cat VERSION 2>/dev/null || echo 0.0.0)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X github.com/AlchemillaHQ/traverse/internal/version.Version=$(VERSION) -X github.com/AlchemillaHQ/traverse/internal/version.Commit=$(COMMIT) -X github.com/AlchemillaHQ/traverse/internal/version.BuildDate=$(DATE)

all: build

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./... -short -count=1 -race -coverprofile=coverage.out

vet:
	go vet ./...

clean:
	rm -f $(BINARY) coverage.out

install:
	go install -ldflags="$(LDFLAGS)" .

version:
	@echo $(VERSION)

bump-patch:
	@$(call bump,patch)

bump-minor:
	@$(call bump,minor)

bump-major:
	@$(call bump,major)

release:
	@echo "Pushing tag v$(VERSION)..."
	git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	git push origin "v$(VERSION)"

release-dry-run:
	goreleaser release --snapshot --clean --skip=publish

help:
	@echo "traverse - STUN/TURN server probe"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the binary"
	@echo "  make test         Run tests with race detector"
	@echo "  make vet          Run go vet"
	@echo "  make clean        Remove build artifacts"
	@echo "  make install      Install via go install"
	@echo "  make version      Print current version"
	@echo ""
	@echo "Version bump (semver):"
	@echo "  make bump-patch   v$(VERSION) -> $$(echo $(VERSION) | awk -F. '{print $$1"."$$2"."$$3+1}')"
	@echo "  make bump-minor   v$(VERSION) -> $$(echo $(VERSION) | awk -F. '{print $$1"."$$2+1".0"}')"
	@echo "  make bump-major   v$(VERSION) -> $$(echo $(VERSION) | awk -F. '{print $$1+1".0.0"}')"
	@echo ""
	@echo "Release:"
	@echo "  make release      Tag and push v$(VERSION)"
	@echo "  make release-dry-run  Local snapshot build (no publish)"
	@echo ""
	@echo "Full release workflow:"
	@echo "  1. Update CHANGELOG.md"
	@echo "  2. make bump-patch   (or bump-minor / bump-major)"
	@echo "  3. git add VERSION CHANGELOG.md && git commit -m 'chore: bump to v$$(cat VERSION)'"
	@echo "  4. make release"

define bump
	@$(eval NEW := $(shell echo $(VERSION) | awk -F. -v part=$(1) \
		'{
			if (part == "major") print $$1+1".0.0";
			else if (part == "minor") print $$1"."$$2+1".0";
			else print $$1"."$$2"."$$3+1
		}'))
	@echo "$(NEW)" > VERSION
	@echo "Bumped: $(VERSION) -> $(NEW)"
endef
