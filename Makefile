.DEFAULT_GOAL := build

SUBDIRS = \
  server/gorilla

TARGETS = fmt vet build clean test test-verbose

.PHONY: $(TARGETS)

$(TARGETS):
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@)
