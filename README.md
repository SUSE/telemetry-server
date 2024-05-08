# telemetry-server
Proof of Concept Telemetry Server scaffolding

To use this code you will need to checkout the 2 telemetry POC related
repos under the same directory:

* github.com/SUSE/telemetry
* github.com/SUSE/telemetry-server

Then you can cd to the telemetry-server/server/gorrila directory and run
the server as follows:
```
% cd telemetry-server/server/gorrila
% go run .
```

Then you can cd to the telemetry/cmd/generator directory and run it to
generate a telemetry data item based upon the content of a specified file,
which will add it to the configured telemetry item data store, as follows:

```
% cd telemetry/cmd/generator
% go run . \
    --config ../../testdata/config/localClient.yaml \
    --tag abc=pqr --tag xyz \
    --telemetry=SLE-SERVER-Test ../../examples/telemetry/SLE-SERVER-Test.json
```
