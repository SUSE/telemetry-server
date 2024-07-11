.DEFAULT_GOAL := build

SUBDIRS = \
  app \
  server/telemetry-server

TARGETS = fmt vet build clean test test-verbose

.PHONY: $(TARGETS)

$(TARGETS):
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@;)
