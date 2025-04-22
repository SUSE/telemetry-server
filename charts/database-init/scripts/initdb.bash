#!/bin/bash

# environmental inputs:
# * TELEMETRY_POSTGRES
#   - hostname/IP of the postgres service
# * POSTGRES_USER
#   - admin user for postgres login
# * PGPASSWORD
#   - admin user's password for postgres login
# * TELEMETRY_PASSWORD
#   - password to setup when creating telemetry user
# * BI_TEAM_PASSWORD
#   - password to setup when creating bi_team user

set -e -u

tmp=/tmp/telemetry
init=init.sql
tmpl=/telemetry/${init}.template

mkdir -p ${tmp}
sed \
	-e "s|__BI_TEAM_PASSWORD__|${BI_TEAM_PASSWORD}|g" -e "s|__TELEMETRY_PASSWORD__|${TELEMETRY_PASSWORD}|g" \
	< ${tmpl} \
	> ${tmp}/${init}

# wait for the postgres server to be available
until pg_isready -h ${TELEMETRY_POSTGRES} -U ${POSTGRES_USER}
do
	sleep 1
done

echo "PostgreSQL is ready, initializing databases..."
psql -h ${TELEMETRY_POSTGRES} -U ${POSTGRES_USER} -f ${tmp}/${init}
echo "PostgreSQL initialization completed successfully."
