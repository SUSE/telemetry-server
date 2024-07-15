FROM registry.suse.com/bci/golang:1.21-openssl AS builder

RUN set -euo pipefail; zypper -n  in --no-recommends git make ; zypper -n clean;

# Create a temporary workspace
WORKDIR /var/cache

# For now, we need this since we use replace directive to point to the local telemetry module in go.mod
RUN git clone https://github.com/SUSE/telemetry
RUN cd telemetry; make build

# Create dest directory for local code
RUN mkdir -p ./telemetry-server/server/telemetry-server

# Copy main go.mod and go.sum to dest directory and run go mod download
COPY go.mod ./telemetry-server
COPY go.sum ./telemetry-server
RUN cd telemetry-server; go mod download

# Copy main go.mod and go.sum to dest directory and run go mod download
COPY server/telemetry-server/go.mod ./telemetry-server/server/telemetry-server
COPY server/telemetry-server/go.sum ./telemetry-server/server/telemetry-server
RUN cd telemetry-server/server/telemetry-server; go mod download

# Copy over only the required contents to run make build
COPY LICENSE Makefile* ./telemetry-server/
COPY app ./telemetry-server/app/
COPY server ./telemetry-server/server/
COPY testdata ./telemetry-server/testdata/

# Build the telemetry server
RUN cd telemetry-server; make build

# Final Image: Start a new build stage with bci-base image as the base and
# copy the built artifacts from the previous stage into this new stage.
FROM registry.suse.com/bci/bci-base:15.6

# Install database support tools
RUN set -euo pipefail; zypper -n install --no-recommends sqlite3 postgresql16; zypper -n clean;

COPY --from=builder /var/cache/telemetry-server/server/telemetry-server/telemetry-server /usr/bin/telemetry-server

ARG cfgFile=dockerServer.yaml
COPY --from=builder /var/cache/telemetry-server/testdata/config/$cfgFile /etc/susetelemetry/server.cfg

#### This block can be removed once we have the package built with a spec that creates user/group/folders
ARG user=tsvc
ARG group=tsvc
ARG uid=1001
ARG gid=1001
RUN mkdir -p /var/lib/${user}
RUN groupadd -g ${gid} ${group}
RUN useradd -r -g ${group} -u ${uid} -d /var/lib/${user} -s /sbin/nologin -c "user for telemetry-server" ${user}
RUN chown -R ${user}:${group} /var/lib/${user}

RUN mkdir -p /tmp/telemetry/server /tmp/susetelemetry
RUN chown -R ${user}:${group} /usr/bin/telemetry-server /tmp/telemetry/server /tmp/susetelemetry


# Put additional files into container
RUN mkdir -p /app
COPY entrypoint.bash /app
RUN chmod 700 /app/entrypoint.bash

ENTRYPOINT ["/app/entrypoint.bash"]
CMD ["--config", "/etc/susetelemetry/server.cfg"]
HEALTHCHECK --interval=30s --timeout=5s CMD curl --fail --insecure http://localhost:9999/healthz || exit 1
