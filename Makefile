SHELL := /bin/bash
.SHELLFLAGS := -ec

GO ?= go
GO_ENV ?= GOWORK=off GOCACHE=$(CURDIR)/.tmp-go-cache GOTMPDIR=$(CURDIR)/.tmp-go-tmp
STATICCHECK ?= go run honnef.co/go/tools/cmd/staticcheck@latest
STATICCHECK_CACHE ?= $(CURDIR)/.tmp-staticcheck-cache

.PHONY: test vet staticcheck race cover verify

test:
	$(GO_ENV) $(GO) test ./...

vet:
	$(GO_ENV) $(GO) vet ./...

staticcheck:
	XDG_CACHE_HOME=$(STATICCHECK_CACHE) $(GO_ENV) GOFLAGS=-buildvcs=false $(STATICCHECK) ./...

race:
	$(GO_ENV) $(GO) test -race -count=1 ./...

cover:
	$(GO_ENV) $(GO) test -cover ./...

verify: test vet staticcheck race
