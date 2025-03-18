# SUSE Telemetry Gateway Server
Implementation for the SUSE Telemetry Gateway service.

NOTE:
Certain actions, such as `make generate` or actions that rely on it,
will require that the github.com/SUSE/telemetry repo is cloned beside
this repo.

The telemetry-server can be run locally or via a docker container, using
either docker compose or docker run directly. Alternatively if podman and
the podman compose commands are available they can be used by passing the
CNTR_MGR=podman argument to the make command when running actions.

NOTE:
There is also a telematry-admin which can be started locally, or via a
docker container, either using docker compose or docker directly.
However the telemetry-admin server doesn't provide any functionality
yet, as the administrative interfaces are still being defined.

## Starting the telemetry-server locally
In a terminal session you can cd to the telemetry-server/server/telemetry-server
directory and run the server as follows:

```
% cd telemetry-server/server/telemetry-server
% rm -rf /tmp/telemetry /tmp/susetelemetry
% mkdir -p /tmp/telemetry/{client,server} /tmp/susetelemetry
% go run . --config ../../testdata/config/localServer.yaml
```

Alternatively you can run `make local-server` in the top-level directory to
launch the local server as shown above. You can optionally specify LOG_LEVEL=debug
variable to enabled debug level logging, as follows:

```
% make local-server LOG_LEVEL=debug
cd server/telemetry-server; \
rm -rf /tmp/telemetry/server
cd server/telemetry-server; \
mkdir -p /tmp/telemetry/server; \
go run . --debug --config ../../testdata/config/localServer.yaml
time=2024-08-12T17:08:11.756-04:00 level=DEBUG msg="Logging initialised" level=DEBUG dest=stderr style=TEXT
time=2024-08-12T17:08:11.756-04:00 level=DEBUG msg="Preparing to start gorilla/mux based server" options="{\"config\":\"../../testdata/config/localServer.yaml\",\"debug\":true}"
time=2024-08-12T17:08:11.756-04:00 level=DEBUG msg="Loading config" path=../../testdata/config/localServer.yaml
...
```

To stop the local server instance you will need to hit Ctrl-C or otherwise
kill the server process.

## Starting the telemetry server with docker compose
Note that the following instructions expect that you have a modern version
of docker with the compose plugin installed.

The docker compose.yaml file is located under the telemetry-server/docker
directory.

Makefile rules are available to run the various docker compose actions

Build the required images:

```
% make compose-build
make -C . vet;  make -C app vet;  make -C server/telemetry-server vet;  make -C server/telemetry-admin vet;
...
cd docker && docker compose build --build-arg telemetryRepoBranch=main
[+] Building 65.2s (73/73) FINISHED                              docker:default
 => [db internal] load build definition from Dockerfile                    0.0s
...
 => CACHED [tsg stage-1 13/13] RUN chmod 700 /app/entrypoint.bash          0.0s
 => [tsg] exporting to image                                               0.0s
 => => exporting layers                                                    0.0s
 => => writing image sha256:9e8c8c9344ee7ebf985791ca54660843173f08e3d9314  0.0s
 => => naming to docker.io/telemetry/server                                0.0s
```

Start the telemetry-server in the background:

```
% make compose-start
...
cd docker && docker compose build
[+] Building 1.9s (73/73) FINISHED                               docker:default
 => [db internal] load build definition from Dockerfile                    0.0s
...
 => => naming to docker.io/telemetry/server                                0.0s
cd docker && docker compose up -d
[+] Running 6/6
 ✔ Network docker_internal  Created                                        0.1s
 ✔ Network docker_external  Created                                        0.2s
 ✔ Volume "docker_pgdata"   Created                                        0.0s
 ✔ Container docker-db-1    Healthy                                        2.9s
 ✔ Container docker-tsa-1   Healthy                                        2.9s
 ✔ Container docker-tsg-1   Started                                        8.4s
```

Check the status of the telemetry services:

```
% make compose-status
cd docker && docker compose ps
NAME           IMAGE                COMMAND                  SERVICE   CREATED              STATUS                        PORTS
docker-db-1    telemetry/postgres   "docker-entrypoint.s…"   db        2 minutes ago   Up 2 minutes (healthy)        5432/tcp
docker-tsa-1   telemetry/admin      "/app/entrypoint.bas…"   tsa       2 minutes ago   Up 2 minutes (healthy)        0.0.0.0:9998->9998/tcp, :::9998->9998/tcp
docker-tsg-1   telemetry/server     "/app/entrypoint.bas…"   tsg       2 minutes ago   Up About a minute (healthy)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp
```

Check the logs for the telemetry-server:

```
% make compose-logs
cd docker && docker compose logs -n 100
...
db-1   | 2024-08-01 14:19:45.222 UTC [1] LOG:  starting PostgreSQL 16.3 (Debian 16.3-1.pgdg120+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 12.2.0-14) 12.2.0, 64-bit
db-1   | 2024-08-01 14:19:45.222 UTC [1] LOG:  listening on IPv4 address "0.0.0.0", port 5432
db-1   | 2024-08-01 14:19:45.222 UTC [1] LOG:  listening on IPv6 address "::", port 5432
db-1   | 2024-08-01 14:19:45.224 UTC [1] LOG:  listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432"
db-1   | 2024-08-01 14:19:45.230 UTC [72] LOG:  database system was shut down at 2024-08-01 14:19:45 UTC
db-1   | 2024-08-01 14:19:45.236 UTC [1] LOG:  database system is ready to accept connections
tsa-1  | time=2024-08-01T14:19:46.005Z level=INFO msg="Database Connected" database=Operational
tsa-1  | time=2024-08-01T14:19:46.030Z level=INFO msg="Database Connected" database=Telemetry
tsa-1  | time=2024-08-01T14:19:46.041Z level=INFO msg="Starting Telemetry Admin" listenOn=tsa:9998
tsg-1  | time=2024-08-01T14:19:52.047Z level=INFO msg="Database Connected" database=Operational
tsg-1  | time=2024-08-01T14:19:52.061Z level=INFO msg="Database Connected" database=Telemetry
tsg-1  | time=2024-08-01T14:19:52.069Z level=INFO msg="Starting Telemetry Server" listenOn=tsg:9999
```

Stop the running telemetry-server:

```
% make compose-stop
cd docker && docker compose ps
NAME           IMAGE                COMMAND                  SERVICE   CREATED         STATUS                   PORTS
docker-db-1    telemetry/postgres   "docker-entrypoint.s…"   db        5 minutes ago   Up 5 minutes (healthy)   5432/tcp
docker-tsa-1   telemetry/admin      "/app/entrypoint.bas…"   tsa       5 minutes ago   Up 5 minutes (healthy)   0.0.0.0:9998->9998/tcp, :::9998->9998/tcp
docker-tsg-1   telemetry/server     "/app/entrypoint.bas…"   tsg       5 minutes ago   Up 5 minutes (healthy)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp
cd docker && docker compose down
[+] Running 5/5
 ✔ Container docker-tsg-1   Removed                                                                                               10.6s 
 ✔ Container docker-tsa-1   Removed                                                                                               10.7s 
 ✔ Container docker-db-1    Removed                                                                                                0.3s 
 ✔ Network docker_external  Removed                                                                                                1.0s 
 ✔ Network docker_internal  Removed                                                                                                0.5s 
```

## Starting the telemetry server with docker run
Makefile rules are available to run the various docker container actions

Build the image:

```
% cd telemetry-server
% make docker-build
docker build -t telemetry-server .
[+] Building 1.3s (33/33) FINISHED                               docker:default
 => [internal] load build definition from Dockerfile                       0.0s
 => => transferring dockerfile: 2.53kB                                     0.0s
 => [internal] load metadata for registry.suse.com/bci/bci-base:15.6       1.1s
 => [internal] load metadata for registry.suse.com/bci/golang:1.21-openss  1.1s
 => [internal] load .dockerignore                                          0.0s
 => => transferring context: 2B                                            0.0s
 => [internal] load build context                                          0.1s
 => => transferring context: 54.93kB                                       0.1s
 => [stage-1  1/13] FROM registry.suse.com/bci/bci-base:15.6@sha256:cc884  0.0s
 => [builder  1/14] FROM registry.suse.com/bci/golang:1.21-openssl@sha256  0.0s
 => CACHED [stage-1  2/13] RUN set -euo pipefail; zypper -n install --no-  0.0s
...
 => CACHED [stage-1 13/13] RUN chmod 700 /app/entrypoint.bash              0.0s
 => exporting to image                                                     0.0s
 => => exporting layers                                                    0.0s
 => => writing image sha256:9a97c35424d6631bee3a0fc2063f51f7c5b418935ea09  0.0s
 => => naming to docker.io/library/telemetry-server                        0.0s
```

Start the docker container:

```
% make docker-start
docker build -t telemetry-server .
[+] Building 1.3s (33/33) FINISHED                               docker:default
...
docker run --rm -it -d -p 9999:9999 --name telemetry-server telemetry-server
676f8dd20e4e04ecea6fcd77efd9fb51daa561d44a66c2a544f8af4f4ecb92ee
```

To check the status of the telemetry-server:

```
% make docker-status
docker ps --filter name=\^telemetry-server\$
CONTAINER ID   IMAGE              COMMAND                  CREATED              STATUS                        PORTS                                       NAMES
676f8dd20e4e   telemetry-server   "/app/entrypoint.bas…"   About a minute ago   Up About a minute (healthy)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp   telemetry-server
```

To check the logs of the telemetry-server:

```
% make docker-logs
docker logs -n 100 telemetry-server
time=2024-07-12T12:15:51.633Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
time=2024-07-12T12:15:51.633Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
time=2024-07-12T12:15:51.638Z level=INFO msg="Database Connected" database=Operational
time=2024-07-12T12:15:51.641Z level=INFO msg="Database Connected" database=Telemetry
time=2024-07-12T12:15:51.653Z level=INFO msg="Starting Telemetry Server" listenOn=0.0.0.0:9999
time=2024-07-12T12:16:21.211Z level=INFO msg=Processing method=GET URL=/healthz
time=2024-07-12T12:16:21.211Z level=INFO msg=Response method=GET URL=/healthz code=200
...
```

To stop the telemetry-server:

```
% make docker-stop
docker ps --filter name=\^telemetry-server\$
CONTAINER ID   IMAGE              COMMAND                  CREATED         STATUS                   PORTS                                       NAMES
676f8dd20e4e   telemetry-server   "/app/entrypoint.bas…"   3 minutes ago   Up 3 minutes (healthy)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp   telemetry-server
docker stop telemetry-server
telemetry-server
```

## Submitting telemetry to the telemetry server
NOTE: Remember to clone the github.com/SUSE/telemetry repo beside this repo
to be able to use the `make generate` action.

You can use the generate Makefile action to submit telemtry as follows:

```
% make generate              
cd ../telemetry/cmd/generator; \
go run . --config ../../testdata/config/localClient.yaml --telemetry=SLE-SERVER-SCCHwInfo --tag DEVTEST ../../testdata/telemetry/SLE-SERVER-SCCHwInfo/sle12sp5-test.json
Generator: config=../../testdata/config/localClient.yaml, dryrun=false, tags=[DEVTEST], telemetry=SLE-SERVER-SCCHwInfo, jsonFiles=[../../testdata/telemetry/SLE-SERVER-SCCHwInfo/sle12sp5-test.json]
2024/07/12 12:15:10 Contents: "telemetry_base_url: http://localhost:9999/telemetry\nenabled: true\ncustomer_id: 1234567890\ntags: []\ndatastores:\n  driver: sqlite3\n  params: /tmp/telemetry/client/telemetry.db\nlogging:\n  level: info\n  location: stderr\n  style: text\n"
2024/07/12 12:15:10 INFO Contents contents="telemetry_base_url: http://localhost:9999/telemetry\nenabled: true\ncustomer_id: 1234567890\ntags: []\ndatastores:\n  driver: sqlite3\n  params: /tmp/telemetry/client/telemetry.db\nlogging:\n  level: info\n  location: stderr\n  style: text\n"
Config: &{TelemetryBaseURL:http://localhost:9999/telemetry Enabled:true CustomerID:1234567890 Tags:[] DataStores:{Driver:sqlite3 Params:/tmp/telemetry/client/telemetry.db} Extras:<nil>}
2024/07/12 12:15:10 INFO NewTelemetryProcessor cfg="&{Driver:sqlite3 Params:/tmp/telemetry/client/telemetry.db}"
2024/07/12 12:15:10 INFO Checking auth file existence authPath=/tmp/susetelemetry/auth.json
2024/07/12 12:15:10 INFO telemetry auth found, client already registered, skipping clientId=1
2024/07/12 12:15:10 Checking size limits for Telemetry Data
2024/07/12 12:15:10 INFO Checking size limits for Telemetry Data "Data size"=302 Max=5242880 Min=10
2024/07/12 12:15:10 INFO Checks passed
2024/07/12 12:15:10 INFO Generated Telemetry name=SLE-SERVER-SCCHwInfo tags=[DEVTEST] content="{\n  \"hostname\": \"sle12sp5-test\",\n  \"distro_target\": \"sle-12-x86_64\",\n  \"hwinfo\": {\n    \"hostname\": \"sle12sp5-test\",\n    \"cpus\": 2,\n    \"sockets\": 1,\n    \"hypervisor\": \"KVM\",\n    \"arch\": \"x86_64\",\n    \"uuid\": \"192653D9-245A-438F-A3F6-4EED1A9C11F3\",\n    \"cloud_provider\": \"\",\n    \"mem_total\": 4096\n  }\n}\n"
2024/07/12 12:15:10 INFO Bundle Tags=[DEVTEST]
2024/07/12 12:15:10 INFO CreateReports Tags=[DEVTEST]
2024/07/12 12:15:10 INFO Checking auth file existence authPath=/tmp/susetelemetry/auth.json
2024/07/12 12:15:10 INFO successfully submitted report report=94dacff0-3424-4259-b575-d6b9d9939e54 processing=0@2024-07-12T16:15:10.332499802Z
```

# Testing
Ensure that you have checked out both telemetry repositories under the
same parent directory and cd into the telemetry-server repo.

## End to End testing
The end to end tests perform the following steps:
* build the docker or docker compose images
* start the server using docker or docker compose
* generate and submit telemetry
  * this will require the github.com/SUSE/telemetry repo to be cloned beside this repo.
* stop the telemetry server using docker or docker compose

The docker end to end test will use an embedded sqlite3 database backend,
while the docker compose tests will deploy postgres database instance.

Run the docker compose end to end tests as follows:

```
% make compose-e2e
cd docker && docker compose build
...
cd docker && docker compose up -d
[+] Running 4/4
 ✔ Network docker_internal  Created                                        0.1s 
 ✔ Network docker_external  Created                                        0.2s 
 ✔ Container docker-db-1    Healthy                                        2.0s 
 ✔ Container docker-tsg-1   Started                                        2.4s 
cd ../telemetry/cmd/generator; \
go run . --config ../../testdata/config/localClient.yaml --telemetry=SLE-SERVER-SCCHwInfo --tag DEVTEST ../../testdata/telemetry/SLE-SERVER-SCCHwInfo/sle12sp5-test.json
...
2024/07/12 12:33:55 INFO successfully submitted report report=05fba282-9606-47e8-b827-dab7dce2ba14 processing=0@2024-07-12T16:33:55.694437735Z
cd docker && docker compose ps
NAME           IMAGE                COMMAND                  SERVICE   CREATED         STATUS                           PORTS
docker-db-1    telemetry/postgres   "docker-entrypoint.s…"   db        4 seconds ago   Up 3 seconds (healthy)           5432/tcp
docker-tsg-1   telemetry/server     "/app/entrypoint.bas…"   tsg       4 seconds ago   Up 1 second (health: starting)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp
cd docker && docker compose down
[+] Running 4/4
 ✔ Container docker-tsg-1   Removed                                       10.7s 
 ✔ Container docker-db-1    Removed                                        0.3s 
 ✔ Network docker_external  Removed                                        0.8s 
 ✔ Network docker_internal  Removed                                        0.5s 
```

Similarly run the docker end to end tests as follows:
```
% make docker-e2e
cd docker && docker compose ps
...
cd docker && docker compose down
...
for cntr in telemetry-server; do \
	docker build -t ${cntr} --target ${cntr} . \
	  --build-arg logLevel=info \
	  --build-arg telemetryCfgDir=/etc/susetelemetry \
		--build-arg telemetryRepoBranch=main \
		--build-arg telemetryImageVariant=upstream; \
done
...
port=9999; \
for cntr in telemetry-server; do \
	docker run --rm -it -d -p ${port}:${port} --name ${cntr} ${cntr}; \
	port=$(expr ${port} - 1); \
done
...
mkdir -p /tmp/susetelemetry
cd ../telemetry/cmd/generator; \
go run .  \
	--config ../../testdata/config/localClient.yaml \
	--telemetry=SLE-SERVER-SCCHwInfo \
	--tag DEVTEST \
	../../testdata/telemetry/SLE-SERVER-SCCHwInfo/sle12sp5-test.json
Added telemetry data from "sle12sp5-test.json" as type "SLE-SERVER-SCCHwInfo" with tags [DEVTEST] to local datastore
Created telemetry bundles from pending telemetry data items
Created telemetry reports from pending telemetry bundles
time=2025-03-27T10:03:35.922-04:00 level=INFO msg="Telemetry Client Registration Required" error="client registration required"
time=2025-03-27T10:03:35.928-04:00 level=INFO msg="Telemetry Client Registration Successful"
Submitted pending telemetry reports
docker ps --filter name=\^$(echo telemetry-server | tr -s ' ' | tr ' ' '|')\$;
...
for cntr in telemetry-server; do \
	docker stop ${cntr}; \
done
...
```

## Running the code validation tests

Run the code validation tests as follows:

```
% cd telemetry-server
% make test
...
```

Or for more verbose output use:

```
% cd telemetry-server
% make test-verbose
...
```

### Running code validation tests using local telemetry repo sources

The above test runs will use the upstream github.com/SUSE/telemetry repo
sources. If you have the telemetry repo cloned beside the telemetry-server
repo, possibly with local changes present, and want to test against those
sources, you can use the local-server-tests make target, which creates a
copy of the two repos beside each under under a /tmp directory, and runs
the tests from there:

```
% cd telemetry-server
% make local-server-tests
...
```
