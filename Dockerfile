#
# Build the code in BCI golang based image
#
FROM registry.suse.com/bci/golang:1.21-openssl AS builder

#
# Git repo settings
#
ARG telemetryRepoName=SUSE/telemetry
ARG telemetryRepoBranch=main

#
# Application names
#
ARG telemetryAdmin=telemetry-admin
ARG telemetryServer=telemetry-server

RUN set -euo pipefail; zypper -n in --no-recommends git make ; zypper -n clean;

# Create a temporary workspace
WORKDIR /var/cache

# For now, we need this since we use replace directive to point to the local telemetry module in go.mod
ADD https://api.github.com/repos/$telemetryRepoName/git/refs/heads/$telemetryRepoBranch telemetry_repo_branch.json
RUN git clone -b $telemetryRepoBranch https://github.com/$telemetryRepoName telemetry
RUN cd telemetry; go mod download -x

# Create dest directories for local code
RUN mkdir -p \
	./telemetry-server/server/$telemetryAdmin \
	./telemetry-server/server/$telemetryServer

# Copy top-level go.mod and go.sum to dest directory and run go mod download
COPY go.mod ./telemetry-server
COPY go.sum ./telemetry-server
RUN cd telemetry-server; go mod download -x

# Copy admin go.mod and go.sum to dest directory and run go mod download
COPY server/$telemetryAdmin/go.mod ./telemetry-server/server/$telemetryAdmin/
COPY server/$telemetryAdmin/go.sum ./telemetry-server/server/$telemetryAdmin/
RUN cd telemetry-server/server/$telemetryAdmin; go mod download -x

# Copy server go.mod and go.sum to dest directory and run go mod download
COPY server/$telemetryServer/go.mod ./telemetry-server/server/$telemetryServer/
COPY server/$telemetryServer/go.sum ./telemetry-server/server/$telemetryServer/
RUN cd telemetry-server/server/$telemetryServer; go mod download -x

# Copy over only the required contents to run make build
COPY LICENSE Makefile* ./telemetry-server/
COPY app ./telemetry-server/app/
COPY server ./telemetry-server/server/
COPY testdata ./telemetry-server/testdata/

# Build the telemetry server
RUN cd telemetry-server; GOFLAGS=-v make build-only

#
# Create the base image that will be used to create the admin
# and service images.
#
FROM registry.suse.com/bci/bci-base:15.6 AS telemetry-base

# Use consistent user and group settings across images
ARG user=tsvc
ARG group=tsvc
ARG uid=1001
ARG gid=1001

# telemetryCfgDir
ARG telemetryCfgDir /etc/susetelemetry

# Install database support tools
RUN set -euo pipefail; zypper -n install --no-recommends sqlite3 postgresql16; zypper -n clean;

#### This block can be removed once we have the package built with a spec that creates user/group/folders
RUN mkdir -p /var/lib/${user}/data
RUN groupadd -g ${gid} ${group}
RUN useradd -r -g ${group} -u ${uid} -d /var/lib/${user} -s /sbin/nologin -c "user for telemetry-server" ${user}
RUN chown -R ${user}:${group} /var/lib/${user}

RUN for d in $telemetryCfgDir /app; do \
		mkdir -p $$d && \
		chown ${user}:${group} $$d || \
		exit 1; \
	done

#
# Create the telemetry-admin image
#
FROM telemetry-base AS telemetry-admin

# some arguement definitions repeated as previous definitions not
# inherited by this image.
ARG telemetryAdmin=telemetry-admin
ARG telemetryCfgDir=/etc/susetelemetry
ARG adminCfg=dockerAdmin.yaml

# copy the built binary from the builder image
COPY --from=builder /var/cache/telemetry-server/server/$telemetryAdmin/$telemetryAdmin /usr/bin/$telemetryAdmin

COPY testdata/config/$adminCfg $telemetryCfgDir/admin.cfg
RUN chown -R ${user}:${group} /usr/bin/$telemetryAdmin

# Put additional files into container
COPY entrypoint-admin.bash /app/entrypoint.bash
RUN chmod 700 /app/entrypoint.bash

ENTRYPOINT ["/app/entrypoint.bash"]
CMD ["--config", "/etc/susetelemetry/admin.cfg"]
HEALTHCHECK --interval=5s --timeout=5s CMD curl --fail --insecure http://localhost:9998/healthz || exit 1

#
# Create the telemetry-server image
#
FROM telemetry-base AS telemetry-server

# some arguement definitions repeated as previous definitions not
# inherited by this image.
ARG telemetryServer=telemetry-server
ARG telemetryCfgDir=/etc/susetelemetry
ARG serverCfg=dockerServer.yaml

# copy the built binary from the builder image
COPY --from=builder /var/cache/telemetry-server/server/$telemetryServer/$telemetryServer /usr/bin/$telemetryServer

COPY testdata/config/$serverCfg /etc/susetelemetry/server.cfg
RUN chown -R ${user}:${group} /usr/bin/$telemetryServer

# Add the container entry point
COPY entrypoint.bash /app/
RUN chmod 700 /app/entrypoint.bash

ENTRYPOINT ["/app/entrypoint.bash"]
CMD ["--config", "/etc/susetelemetry/server.cfg"]
HEALTHCHECK --interval=5s --timeout=5s CMD curl --fail --insecure http://localhost:9999/healthz || exit 1
