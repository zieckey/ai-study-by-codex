.PHONY: test run clean-cache

# macOS 26 + Go 1.22 internal-link cgo binaries can fail with:
#   dyld: missing LC_UUID load command
# This project does not need cgo, so use pure-Go resolver/user lookup by default.
GOFLAGS ?= -tags=netgo,osusergo
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/go-mod
GOTMPDIR ?= $(CURDIR)/.cache/go-tmp

export GOFLAGS GOCACHE GOMODCACHE GOTMPDIR

test:
	@mkdir -p $(GOCACHE) $(GOMODCACHE) $(GOTMPDIR)
	go test ./...

run:
	@mkdir -p $(GOCACHE) $(GOMODCACHE) $(GOTMPDIR)
	go run ./cmd/agent --trace "$(q)"

clean-cache:
	go clean -cache -testcache
