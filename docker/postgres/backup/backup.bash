#!/bin/bash
set -euo pipefail

echo "Starting backup job..."

DBS=(operational telemetry)

for DB in "${DBS[@]}"; do
    echo "Backing up $DB..."
    BACKUP_FILE="/backup/${DB}-$(date +%Y%m%d-%H%M%S).sql.gz"
    pg_dump -h telemetry-postgres -U "$POSTGRES_USER" "$DB" | gzip > "$BACKUP_FILE"
    echo "Uploading $BACKUP_FILE to S3..."
    aws s3 cp "$BACKUP_FILE" "s3://$AWS_BUCKET/backups/$DB/$(basename "$BACKUP_FILE")"
done

echo "All backups completed successfully."
