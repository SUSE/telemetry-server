# telemetry-server
This is the basic implementation of the SUSE Telemetry Gateway service.

To use this code you will need to checkout both telemetry repositories
under the same directory:

* github.com/SUSE/telemetry
* github.com/SUSE/telemetry-server

In one terminal session you can cd to the telemetry-server/server/telemetry-server
directory and run the server as follows:

```
% cd telemetry-server/server/telemetry-server
% rm -rf /tmp/telemetry /tmp/susetelemetry
% mkdir -p /tmp/telemetry/{client,server} /tmp/susetelemetry 
% go run . --config ../../testdata/config/localServer.yaml
```

The telemetry-server can be run as a docker container.
Build the image:
```
% cd telemetry-server
% docker build -t telemetry-server .
```
Run the docker container:
NOTE:  --network=host is used only for local docker run based testing
```
% docker run --network=host --rm -it -d -p 9999:9999 --name telemetry-server telemetry-server
```

Then in another terminal session you can cd to telemetry/cmd/generator
directory and run the client tool to generate a telemetry data item
based upon the content of a specified file, which will add it to the
configured telemetry item data store, as follows:

```
% cd telemetry/cmd/generator
% go run . \
    --config ../../testdata/config/localClient.yaml \
    --tag abc=pqr --tag xyz \
    --telemetry=SLE-SERVER-Test ../../examples/telemetry/SLE-SERVER-Test.json
```

# testing

Ensure that you have checked out both telemetry repositories under the
same directory and then cd into the telemetry-server repo and run the
as follows:

```
% cd telemetry-server
% make test
```
