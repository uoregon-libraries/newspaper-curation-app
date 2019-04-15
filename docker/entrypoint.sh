#!/usr/bin/env bash
set -eu

if [[ ${NCA_NEWS_WEBROOT:-} == "" ]]; then
  echo "NCA_NEWS_WEBROOT must be set"
  exit 1
fi

echo "Waiting for database connectivity"
wait_for_database

echo "Running migrations"
lockfile=/mnt/news/goose-running
flock $lockfile -c "goose up"

echo "Ensuring directories are present"
source settings && mkdir -p $MASTER_PDF_UPLOAD_PATH
source settings && mkdir -p $MASTER_SCAN_UPLOAD_PATH
source settings && mkdir -p $MASTER_PDF_BACKUP_PATH
source settings && mkdir -p $PDF_PAGE_REVIEW_PATH
source settings && mkdir -p $BATCH_OUTPUT_PATH
source settings && mkdir -p $WORKFLOW_PATH

echo 'Executing "'$@'"'
cd /usr/local/nca
exec $@
