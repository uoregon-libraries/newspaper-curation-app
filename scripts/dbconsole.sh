#!/bin/env bash
set -eu

source settings
mysql -u $DB_USER -D $DB_DATABASE -h $DB_HOST -p$DB_PASSWORD
