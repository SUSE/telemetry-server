#
# Go Environment Settings
#
ARG GO_NO_PROXY=github.com/SUSE

#
# PostgreSQL Settings
#
ARG POSTGRES_BASE=registry.suse.com/suse/postgres
ARG POSTGRES_MAJOR=16

#
# Telemetry Build Settings
#
ARG telemetryImageVariant=upstream
ARG telemetryRepoName=SUSE/telemetry
ARG telemetryRepoBranch=main
ARG telemetryCacheDir=/var/cache

#
# Telemetry Service Settings
#
ARG telemetryCfgDir=/etc/susetelemetry
ARG adminCfg=dockerAdmin.yaml
ARG serverCfg=dockerServer.yaml
ARG logLevel=info

# use consistent user and group settings across images
ARG user=tsvc
ARG group=tsvc
ARG uid=1001
ARG gid=1001

#
# Telemetry Service Application names
#
ARG telemetryAdmin=telemetry-admin
ARG telemetryServer=telemetry-server

#
# Build the customised telemetry-postgres image
#
FROM ${POSTGRES_BASE}:${POSTGRES_MAJOR} AS telemetry-postgres

# add the telemetry init scripting
ADD docker/postgres/telemetry /telemetry
RUN chmod +x /telemetry/*.bash

#
# Build the code in BCI golang based image
#
FROM registry.suse.com/bci/golang:1.23-openssl AS builder-base

# these args are used in this stage
ARG telemetryCacheDir
ARG telemetryAdmin
ARG telemetryServer

RUN set -euo pipefail; \
	zypper -n install --no-recommends git make; \
	zypper -n clean

# create a temporary workspace
WORKDIR ${telemetryCacheDir}

# create dest directories for local code
RUN mkdir -p \
	./telemetry-server/app \
	./telemetry-server/server/${telemetryAdmin} \
	./telemetry-server/server/${telemetryServer}

# Copy over top-level telemetry-server repo go.mod and go.sum files
COPY go.mod ./telemetry-server/
COPY go.sum ./telemetry-server/

#
# Create an intermediate builder image with the telemetry repo cloned
# on the specified branch or tag
#
FROM builder-base AS builder-telemetry-specified

# these args are used in this stage
ARG GO_NO_PROXY
ARG telemetryRepoBranch
ARG telemetryRepoName

# retrieve the branch or tag hash before attempting to clone the repo to
# ensure repo will be freshly cloned if it's content changes
#ONBUILD ADD https://api.github.com/repos/${telemetryRepoName}/git/refs/heads/${telemetryRepoBranch} telemetry_repo_branch.json
ONBUILD RUN \
        if curl --silent --fail \
		https://api.github.com/repos/${telemetryRepoName}/git/refs/heads/${telemetryRepoBranch} \
		> telemetry_repo_branch.json; then \
		echo Saved branch hash data; \
	elif curl --silent --fail \
		https://api.github.com/repos/${telemetryRepoName}/git/refs/tags/${telemetryRepoBranch} \
		> telemetry_repo_branch.json; then \
		echo Saved tag hash data; \
	else \
		echo ${telemetryRepoBranch} is neither a branch or a tag; \
	fi; \
	[[ -s telemetry_repo_branch.json ]] || exit 1


# clone telemetry repo on desired branch or tag beside telemetry-server,
# download the required mods, and then update telemetry-server repo to
# use cloned telemetry repo as it's source.
ONBUILD RUN \
	head /dev/null telemetry_repo_branch.json; \
	set -euo pipefail; \
	echo Building with named telemetry repo branch ${telemetryRepoBranch} && \
	export GONOPROXY=${GO_NO_PROXY} && \
	git clone \
		--branch ${telemetryRepoBranch} \
		https://github.com/${telemetryRepoName} \
		telemetry && \
	cd telemetry && \
	go mod download -x && \
	cd ../telemetry-server && \
	go mod edit --replace \
		github.com/SUSE/telemetry=../telemetry/ && \
	go mod tidy -x

#
# Create an intermediate builder image without the telemetry repo cloned
#
FROM builder-base AS builder-telemetry-upstream
ONBUILD RUN echo Building with upstream telemetry sources

#
# Create a builder image based upon the appropriate intermediate builder image
#
FROM builder-telemetry-${telemetryImageVariant} AS builder

# these args are used in this stage
ARG GO_NO_PROXY
ARG telemetryAdmin
ARG telemetryServer

# run go mod download for top-level directory
RUN cd telemetry-server; \
	env GONOPROXY=${GO_NO_PROXY} go mod download -x

# Copy admin go.mod and go.sum to dest directory and run go mod download
COPY server/${telemetryAdmin}/go.mod ./telemetry-server/server/${telemetryAdmin}/
COPY server/${telemetryAdmin}/go.sum ./telemetry-server/server/${telemetryAdmin}/
RUN cd telemetry-server/server/${telemetryAdmin}; \
	env GONOPROXY=${GO_NO_PROXY} go mod download -x

# Copy server go.mod and go.sum to dest directory and run go mod download
COPY server/${telemetryServer}/go.mod ./telemetry-server/server/${telemetryServer}/
COPY server/${telemetryServer}/go.sum ./telemetry-server/server/${telemetryServer}/
RUN cd telemetry-server/server/${telemetryServer}; \
	env GONOPROXY=${GO_NO_PROXY} go mod download -x

# Copy over only the required contents to run make build
COPY LICENSE Makefile* ./telemetry-server/
COPY app ./telemetry-server/app/
COPY server ./telemetry-server/server/

# Build the telemetry server, running make mod-tidy as needed if a local
# telemetry repo is being used
RUN cd telemetry-server; \
	if [[ -d ../telemetry ]]; then \
		env GONOPROXY=${GO_NO_PROXY} make mod-tidy; \
	fi; \
	env GOFLAGS=-v make build-only

#
# Create the base image that will be used to create the admin
# and service images.
#
FROM registry.suse.com/bci/bci-base:15.6 AS telemetry-base

# these args are used in this stage
ARG telemetryAdmin
ARG telemetryServer
ARG telemetryCfgDir
ARG user
ARG uid
ARG group
ARG gid

# Install database support tools
RUN set -euo pipefail; zypper -n install --no-recommends sqlite3 postgresql16 iproute2; zypper -n clean;

#### This block can be removed once we have the package built with a spec that creates user/group/folders
RUN mkdir -p /var/lib/${user}/data
RUN groupadd -g ${gid} ${group}
RUN useradd -r -g ${group} -u ${uid} -d /var/lib/${user} -s /sbin/nologin -c "user for telemetry-server" ${user}
RUN chown -R ${user}:${group} /var/lib/${user}

RUN for d in ${telemetryCfgDir} /app; do \
		mkdir -p $$d && \
		chown ${user}:${group} $$d || \
		exit 1; \
	done

COPY entrypoint.bash /app/
RUN chmod 700 /app/entrypoint.bash

#
# Create the telemetry-admin image
#
FROM telemetry-base AS telemetry-admin

# these args are used in this stage
ARG telemetryAdmin
ARG telemetryCfgDir
ARG logLevel
ARG user
ARG group
ARG adminCfg

# copy the built binary from the builder image
COPY --from=builder /var/cache/telemetry-server/server/${telemetryAdmin}/${telemetryAdmin} /usr/bin/${telemetryAdmin}
RUN chown -R ${user}:${group} /usr/bin/${telemetryAdmin}

# copy over the config file
COPY testdata/config/$adminCfg $telemetryCfgDir/admin.cfg

# Put additional files into container
RUN echo "TELEMETRY_SERVICE=${telemetryAdmin}" > /etc/default/susetelemetry
RUN echo "LOG_LEVEL=${logLevel}" >> /etc/default/susetelemetry

ENTRYPOINT ["/app/entrypoint.bash"]
CMD ["--config", "/etc/susetelemetry/admin.cfg"]
HEALTHCHECK --interval=5s --timeout=5s CMD curl --fail --insecure http://localhost:9998/healthz || exit 1

#
# Create the telemetry-server image
#
FROM telemetry-base AS telemetry-server

# these args are used in this stage
ARG telemetryServer
ARG telemetryCfgDir
ARG logLevel
ARG user
ARG group
ARG serverCfg

# copy the built binary from the builder image
COPY --from=builder /var/cache/telemetry-server/server/${telemetryServer}/${telemetryServer} /usr/bin/${telemetryServer}
RUN chown -R ${user}:${group} /usr/bin/$telemetryServer

# copy over the config file
COPY testdata/config/${serverCfg} ${telemetryCfgDir}/server.cfg

# Add the container entry point
RUN echo "TELEMETRY_SERVICE=${telemetryServer}" > /etc/default/susetelemetry
RUN echo "LOG_LEVEL=${logLevel}" >> /etc/default/susetelemetry

ENTRYPOINT ["/app/entrypoint.bash"]
CMD ["--config", "/etc/susetelemetry/server.cfg"]
HEALTHCHECK --interval=5s --timeout=5s CMD curl --fail --insecure http://localhost:9999/healthz || exit 1
