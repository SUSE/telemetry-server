module github.com/SUSE/telemetry-server/app

go 1.21

replace github.com/SUSE/telemetry-server => ../../telemetry-server/

require github.com/mattn/go-sqlite3 v1.14.22

require (
	github.com/SUSE/telemetry v0.0.0-00010101000000-000000000000
	github.com/go-playground/validator/v10 v10.21.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/xyproto/randomstring v1.0.5 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/SUSE/telemetry => ../../telemetry/

replace github.com/SUSE/telemetry/pkg/config => ../../telemetry/pkg/config

replace github.com/SUSE/telemetry/pkg/lib => ../../telemetry/pkg/lib
