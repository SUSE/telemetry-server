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

# Loop through all (db, table) pairs from the JSON representation of dbs and tables to check ($DB_CHECKS)
echo "${DB_CHECKS}" | jq -r 'to_entries[] | "\(.key) \(.value[])"' | while read -r db table; do
    check_table "${db}" "${table}"
done

echo "All database checks passed successfully"
