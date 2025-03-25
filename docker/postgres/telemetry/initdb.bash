#!/bin/bash

set -e -u

init=/telemetry/init.sql
tmpl=${init}.template

sed \
	-e "s|__BI_TEAM_PASSWORD__|${BI_TEAM_PASSWORD}|g" -e "s|__TELEMETRY_PASSWORD__|${TELEMETRY_PASSWORD}|g" \
	< ${tmpl} \
	> ${init}

until pg_isready -h ${RDS_HOST} -U ${POSTGRES_USER}
do
	sleep 1
done

psql -h ${RDS_HOST} -U ${POSTGRES_USER} -f ${init}
