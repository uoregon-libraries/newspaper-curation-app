#!/usr/bin/env bash
set -eu

EXPORT_DIR="./mysql-export"

settings=${1:-./settings}
echo "Attempting to read NCA's settings from $settings"
source $settings

# Create export directory if it doesn't exist
mkdir -p "$EXPORT_DIR"

# Function to perform mysqldump with arguments
function dump_table() {
  local table="$1"
  local where="${2:-}"
  mysqldump --host="$DB_HOST" --user="$DB_USER" --password="$DB_PASSWORD" --databases "$DB_DATABASE" \
    --no-create-db --no-create-info --skip-triggers \
    --tables "$table" \
    ${where:+"--where=$where"} \
    > "$EXPORT_DIR/99-data-$table.sql"
}

rm -f $EXPORT_DIR/*.sql
rm -f $EXPORT_DIR/*.sql.gz

echo "Exporting database structure..."
mysqldump --host="$DB_HOST" --user="$DB_USER" --password="$DB_PASSWORD" --databases "$DB_DATABASE" --no-data --routines --triggers > "$EXPORT_DIR/00-structure.sql"

dt=$(date -d "1 month ago" +"%Y-%m-%d")
echo "Exporting table data..."
for table in $(mysql --host="$DB_HOST" --user="$DB_USER" --password="$DB_PASSWORD" -s -e "SHOW TABLES FROM $DB_DATABASE")
do
    echo "Exporting table: $table"
    if [[ $table == "audit_logs" ]]; then
        dump_table "$table" "\`when\` > '$dt'"
    elif [[ $table == "pipelines" || $table == "jobs" || $table == "job_logs" ]]; then
        dump_table "$table" "created_at > '$dt'"
    else
        dump_table "$table"
    fi
done

echo "Combining all sql into a gzipped archive"
for file in $(find $EXPORT_DIR -type f | sort); do
  cat $file >> $EXPORT_DIR/export.sql
done
gzip $EXPORT_DIR/export.sql

echo "MySQL data export complete. Files saved in $EXPORT_DIR."
