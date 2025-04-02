#!/bin/sh

# environmental inputs:
# * TELEMETRY_POSTGRESS
# * TELEMETRY_USER
# * TELEMETRY_PASSWORD

set -eu
set -o pipefail

echo "Checking PostgreSQL connectivity"

psql_cmd()
{
    local db="${1}" result
    shift

    env PGPASSWORD=${TELEMETRY_PASSWORD} \
        psql \
            -h ${TELEMETRY_POSTGRES} \
            ${db:+-d ${db}} \
            -U "${TELEMETRY_USER}" \
            -t \
            -c "${*}"
}

check_db_exists()
{
    local db="${1}" result

    result=$(psql_cmd "" "SELECT datname FROM pg_catalog.pg_database WHERE datname = '${db}';" | tr -d ' ')

    [[ "${result}" == "${db}" ]]
}

check_table_exists()
{
    local db="${1}" table="${2}" result

    result=$(psql_cmd "${db}" "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = '${table}';" | tr -d ' ')

    [[ "${result}" == "${table}" ]]
}

declare -A databases
databases[operational]="clients reports"
databases[telemetry]="customers tagsets telemetrydata"

exit_status=0
for db in "${!databases[@]}"; do
    echo "Checking database: ${db}"
    if ! check_db_exists "${db}"; then
        echo "ERROR: Database '${db}' missing!"
        exit_status=1
        continue
    fi

    for table in ${databases[${db}]}
    do
        echo "  Checking table: ${table}"
        if ! check_table_exists "${db}" "${table}"; then
            echo "WARNING: Database '${db}' table '${table}' missing - has telemetry service been run yet?"
            continue
        fi
    done
done

if (( exit_status )); then
    exit ${exit_status}
fi

echo "All database checks passed successfully"

# vim:shiftwidth=4:tabstop=4:expandtab
