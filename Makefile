GO ?= go

.PHONY: build run dev release install fmt vet test check smoke changelog

build: ; $(GO) build ./...
run: ; $(GO) run ./cmd/mremoteng
dev: ; ./scripts/dev.sh
release: ; ./scripts/release.sh
install: ; ./scripts/release.sh --install
fmt: ; gofmt -w .
vet: ; $(GO) vet ./...
test: ; $(GO) test ./...
check: ; ./scripts/check.sh
smoke: ; ./scripts/smoke.sh
changelog: ; $(GO) run ./cmd/changelog compile
