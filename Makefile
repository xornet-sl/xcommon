# Required for globs to work correctly
SHELL           = /usr/bin/env bash

TESTS           := .
TESTFLAGS       := -race -v

GOFLAGS         :=

GIT_COMMIT      = $(shell git rev-parse HEAD 2>/dev/null)
GIT_SHA         = $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_TAG         = $(shell git describe --tags --abbrev=0 --match='v*' --candidates=1 2>/dev/null)
GIT_STATUS      = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

ifeq ($(GIT_SHA),)
GIT_STATUS      = invalid
endif

ifneq ($(GIT_TAG),)
VERSION         ?= $(GIT_TAG)
else
VERSION         ?= "dev"
endif


.PHONY: test
test:
	@go test $(GOFLAGS) -run $(TESTS) ./... $(TESTFLAGS)


.PHONY: info
info:
	 @echo "Version:           $(VERSION)"
	 @echo "Git Tag:           $(GIT_TAG)"
	 @echo "Git Commit:        $(GIT_COMMIT)"
	 @echo "Git Tree Status:   $(GIT_STATUS)"
