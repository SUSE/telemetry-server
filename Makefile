.DEFAULT_GOAL := build
LOG_LEVEL = info
CNTR_MGR = docker
TELEMETRY_REPO_BRANCH ?= main

include Makefile.local-server
include Makefile.compose
include Makefile.docker
include Makefile.generate
include Makefile.e2e
include Makefile.golang
