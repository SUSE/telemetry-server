.DEFAULT_GOAL := build

SUBDIRS = \
  app \
  server/telemetry-server

TARGETS = fmt vet build clean test test-verbose
COMPOSE_TARGETS = compose-build compose-start compose-up compose-ps compose-status compose-logs compose-stop compose-down
DOCKER_TARGETS = docker-build docker-start docker-run docker-ps docker-status docker-logs docker-stop

.PHONY: $(TARGETS) $(COMPOSE_TARGETS) $(DOCKER_TARGETS)

$(TARGETS):
	$(foreach subdir, $(SUBDIRS), $(MAKE) -C $(subdir) $@;)

# Start the telemetry-server using docker compose
compose-build:
	cd docker && docker compose build

compose-start compose-up: compose-build
	cd docker && docker compose up -d

compose-ps compose-status:
	cd docker && docker compose ps

compose-logs:
	cd docker && docker compose logs -n 100

compose-stop compose-down: compose-status
	cd docker && docker compose down

# Start the telemetry-server using docker
docker-build:
	docker build -t telemetry-server .

docker-start docker-run: docker-build
	docker run --rm -it -d -p 9999:9999 --name telemetry-server telemetry-server

docker-ps docker-status:
	docker ps --filter name=\^telemetry-server\$

docker-logs:
	docker logs -n 100 telemetry-server

docker-stop: docker-status
	docker stop telemetry-server
