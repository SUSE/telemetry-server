#!/bin/sh
set -eu
set -o pipefail

echo "Checking PostgreSQL connectivity"

check_table() {
    local db=$1
    local table=$2
    local result=$(env PGPASSWORD=${TELEMETRY_PASSWORD} psql -h ${RDS_HOST} -U "${TELEMETRY_USER}" -d "${db}" -t -c "SELECT to_regclass('${table}');" | tr -d \ \'\" )

    if [[ ${result} == ${table} ]]; then
        echo "Table '${table}' exists in database '${db}'"
    else
        echo "Table '${table}' is missing in database '${db}'"
        exit 1
    fi
}

declare -A databases
databases[operational]="clients reports"
databases[telemetry]="customers tagsets telemetrydata"

for db in "${!databases[@]}"; do
    echo "Checking database: ${db}"
    
    for table in ${databases[${db}]}; do
        check_table "${db}" "${table}"
    done
done

echo "All database checks passed successfully"
