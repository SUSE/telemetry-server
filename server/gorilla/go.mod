module github.com/SUSE/telemetry-server/server/gorilla

go 1.21

replace github.com/SUSE/telemetry => ../../../telemetry/

replace github.com/SUSE/telemetry-server => ../../../telemetry-server/

require (
	github.com/SUSE/telemetry-server/app v0.0.0-00010101000000-000000000000
	github.com/go-playground/validator/v10 v10.21.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/xyproto/randomstring v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/SUSE/telemetry v0.0.0-00010101000000-000000000000

require (
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)

replace github.com/SUSE/telemetry-server/app => ../../../telemetry-server/app/
