include Makefile.compose
include Makefile.docker
include Makefile.generate

.DEFAULT_GOAL := build

SUBDIRS = \
  app \
  server/telemetry-server

TARGETS = fmt vet build clean test test-verbose

.PHONY: $(TARGETS)

$(TARGETS):
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@;)


.PHONY: end-to-end e2e

end-to-end e2e: compose-start generate compose-stop
