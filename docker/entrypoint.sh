#!/usr/bin/env bash
set -eu
echo "Waiting for database connectivity"
wait_for_database

echo 'Executing "'$@'"'
cd /usr/local/black-mamba
exec $@
