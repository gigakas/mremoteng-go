GO ?= go

.PHONY: build fmt vet test check smoke changelog

build: ; $(GO) build ./...
fmt: ; gofmt -w .
vet: ; $(GO) vet ./...
test: ; $(GO) test ./...
check: ; ./scripts/check.sh
smoke: ; ./scripts/smoke.sh
changelog: ; $(GO) run ./cmd/changelog compile
