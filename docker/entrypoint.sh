#!/usr/bin/env bash
set -eu

if [[ ${APP_URL:-} == "" ]]; then
  echo "APP_URL environment variable is not set; if the application won't"
  echo "start due to an invalid WEBROOT, this could be why"
  echo
fi
if [[ ${NCA_NEWS_WEBROOT:-} == "" ]]; then
  echo "NCA_NEWS_WEBROOT must be set"
  exit 1
fi

echo "Waiting for database connectivity"
wait_for_database

echo 'Executing "'$@'"'
cd /usr/local/black-mamba
exec $@
