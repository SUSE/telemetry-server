module github.com/SUSE/telemetry-server/server/telemetry-admin

go 1.21

replace github.com/SUSE/telemetry-server => ../../../telemetry-server/

replace github.com/SUSE/telemetry => ../../../telemetry/

require (
	github.com/go-playground/validator/v10 v10.22.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1
	github.com/stretchr/testify v1.9.0
	github.com/xyproto/randomstring v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

require (
	github.com/SUSE/telemetry v0.0.0-20240724131408-7d6373a5dac3
	github.com/SUSE/telemetry-server v0.0.0-20240614161816-bafbd5826391
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)
