module github.com/SUSE/telmetry-server/app

go 1.21

replace github.com/SUSE/telmetry-server => ../../telemetry-server/

require github.com/mattn/go-sqlite3 v1.14.22

require (
	github.com/SUSE/telemetry v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/xyproto/randomstring v1.0.5 // indirect
)

replace github.com/SUSE/telemetry => ../../telemetry/

replace github.com/SUSE/telemetry/pkg/config => ../../telemetry/pkg/config

replace github.com/SUSE/telemetry/pkg/lib => ../../telemetry/pkg/lib
