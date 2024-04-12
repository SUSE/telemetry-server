module github.com/SUSE/telmetry-server/app

go 1.21

replace github.com/SUSE/telmetry-server => ../../telemetry-server/

require github.com/mattn/go-sqlite3 v1.14.22

require github.com/SUSE/telemetry v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/SUSE/telemetry => ../../telemetry/

replace github.com/SUSE/telemetrylib => ../../telemetrylib/
