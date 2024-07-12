# SUSE Telemetry Gateway Server
Implementation for the SUSE Telemetry Gateway service.

To use this code you will need to checkout both telemetry repositories
under the same directory:

* github.com/SUSE/telemetry
* github.com/SUSE/telemetry-server

The telemetry-server can be run locally or via a docker container, using
either docker compose or docker run directly.

## Starting the telemetry-server locally
In a terminal session you can cd to the telemetry-server/server/telemetry-server
directory and run the server as follows:

```
% cd telemetry-server/server/telemetry-server
% rm -rf /tmp/telemetry /tmp/susetelemetry
% mkdir -p /tmp/telemetry/{client,server} /tmp/susetelemetry
% go run . --config ../../testdata/config/localServer.yaml
```

## Starting the telemetry server with docker compose
NOTE: Makefile rules are available to run the various docker compose actions

First cd into the telemetry-server/docker directory:
```
% cd telemetry-server/docker
```

Note that the following instructions expect that you have a modern version
of docker with the compose plugin installed.

Build the required images:
```
% docker compose build
[+] Building 16.2s (33/33) FINISHED                              docker:default
 => [tsg internal] load build definition from Dockerfile                   0.0s
 => => transferring dockerfile: 2.41kB                                     0.0s
...
 => [tsg stage-1 13/13] RUN chmod 700 /app/entrypoint.bash                 0.4s
 => [tsg] exporting to image                                               0.2s
 => => exporting layers                                                    0.2s
 => => writing image sha256:013838f5bc8ca8de35891abfb14eaf61540f6f5788a30  0.0s
 => => naming to docker.io/telemetry/server                                0.0s
```

Start the telemetry-server in the background:
```
% docker compose up -d
[+] Running 2/2
 ✔ Network docker_external  Created                                        0.1s
 ✔ Container docker-tsg-1   Started                                        0.4s
```

Check the status of the telemetry-server container:
```
% docker compose ps
NAME           IMAGE              COMMAND                  SERVICE   CREATED         STATUS                            PORTS
docker-tsg-1   telemetry/server   "/app/entrypoint.bas…"   tsg       7 seconds ago   Up 6 seconds (health: starting)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp
```

Check the logs for the telemetry-server:
```
% docker compose logs
tsg-1  | time=2024-07-11T20:30:42.645Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
tsg-1  | time=2024-07-11T20:30:42.646Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
tsg-1  | time=2024-07-11T20:30:42.664Z level=INFO msg="Starting Telemetry Server" listenOn=tsg:9999
tsg-1  | time=2024-07-11T20:31:12.314Z level=INFO msg=Processing method=GET URL=/healthz
tsg-1  | time=2024-07-11T20:31:12.314Z level=INFO msg=Response method=GET URL=/healthz code=200
...
```

Stop the running telemetry-server:
```
% docker compose down
[+] Running 2/2
 ✔ Container docker-tsg-1   Removed                                       10.4s
 ✔ Network docker_external  Removed                                        0.4s
```

## Starting the telemetry server with docker run
NOTE: Makefile rules are available to run the various docker container actions

Build the image:
```
% cd telemetry-server
% docker build -t telemetry-server .
```
Run the docker container:
```
% docker run --rm -it -d -p 9999:9999 --name telemetry-server telemetry-server
bbb048cbfa84fff4e7455c46c028963dbbdfbf70ed8033f9abe3dd7715603c72
```

To check the status of the telemetry-server:
```
% docker ps
CONTAINER ID   IMAGE              COMMAND                  CREATED              STATUS                        PORTS                                       NAMES
bbb048cbfa84   telemetry-server   "/app/entrypoint.bas…"   About a minute ago   Up About a minute (healthy)   0.0.0.0:9999->9999/tcp, :::9999->9999/tcp   telemetry-server
```

To check the logs of the telemetry-server:
```
% docker logs telemetry-server
time=2024-07-11T21:09:26.407Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
time=2024-07-11T21:09:26.407Z level=INFO msg="Logging initialised" level=INFO dest=stderr style=TEXT
time=2024-07-11T21:09:26.423Z level=INFO msg="Starting Telemetry Server" listenOn=0.0.0.0:9999
time=2024-07-11T21:09:56.026Z level=INFO msg=Processing method=GET URL=/healthz
time=2024-07-11T21:09:56.026Z level=INFO msg=Response method=GET URL=/healthz code=200
...
```

To stop the telemetry-server:
```
% docker stop telemetry-server
```

## Submitting telemetry to the telemetry server
In another terminal session you can cd to telemetry/cmd/generator
directory and run the generator tool to submit a telemetry data item
based upon the content of the specified JSON blob file, which will
add it to the configured telemetry item data store and submit it to
the telemetry server, as follows:

```
% cd telemetry/cmd/generator
% go run . \
    --config ../../testdata/config/localClient.yaml \
    --tag abc=pqr --tag xyz \
    --telemetry=SLE-SERVER-Test ../../examples/telemetry/SLE-SERVER-Test.json
```

# Testing

Ensure that you have checked out both telemetry repositories under the
same directory and then cd into the telemetry-server repo and run the
as follows:

```
% cd telemetry-server
% make test
```
