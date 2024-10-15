#!/usr/bin/env bash
set -eu

if [[ ${NCA_NEWS_WEBROOT:-} == "" ]]; then
  echo "NCA_NEWS_WEBROOT must be set"
  exit 1
fi

echo "Waiting for database connectivity"
wait_for_database

echo "Running migrations"
lockfile=/mnt/news/migrations-running
flock $lockfile -c "./bin/migrate-database -c ./settings up"

echo "Get SFTPgo admin API key and store in NCA settings file"
flock /mnt/news/get-sftpgo-api-key-running -c "SETTINGS_PATH=settings SFTPGO_ADMIN_LOGIN=admin SFTPGO_ADMIN_PASSWORD=password sftpgo/get_admin_api_key.sh --force"

echo 'Executing "'$@'"'
cd /usr/local/nca
exec $@
