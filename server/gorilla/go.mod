module github.com/SUSE/telemetry-server/server/gorilla

go 1.22.0

replace github.com/SUSE/telemetry => ../../../telemetry

replace github.com/SUSE/telemetry-server => ../../

require (
	github.com/SUSE/telemetry-server/app v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.8.1
)

require github.com/mattn/go-sqlite3 v1.14.22 // indirect

replace github.com/SUSE/telemetry-server/app => ../../../telemetry-server/app
