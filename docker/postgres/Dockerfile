# Use the latest upstream postgres:16 image
FROM postgres:16

# install our initdb.sql script so that it will be run during startup
COPY initdb.sql /docker-entrypoint-initdb.d/0000-initdb.sql
