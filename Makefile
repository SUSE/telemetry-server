export GO_COVERAGE_PROFILE = /tmp/.coverage.telemetry-server.out

ifeq ($(MAKELEVEL),0)

LOG_LEVEL=info
CNTR_MGR = docker
TELEMETRY_REPO_BRANCH = main

include Makefile.local-server
include Makefile.compose
include Makefile.docker
include Makefile.generate
include Makefile.e2e

.DEFAULT_GOAL := build

SUBDIRS = \
  . \
  app \
  server/telemetry-server \
  server/telemetry-admin

TARGETS = fmt vet build build-only clean test test-clean test-verbose tidy

.PHONY: $(TARGETS)

$(TARGETS)::
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@ || exit 1;)

else
include Makefile.golang
endif
