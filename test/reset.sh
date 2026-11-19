#!/usr/bin/env bash
#
# Quick reset between test runs: clears issues, jobs, and batches from NCA's
# database and reloads test/fakemount from the test sources, but keeps users,
# titles, and the cached live-site data, which take much longer to rebuild.
set -eu

cd "$(dirname "$0")/.."
source scripts/localdev.sh

nca_compose stop web workers
ncasql -e "delete from jobs; delete from issues; delete from job_logs; delete from batches;"

# Remove the finder cache, but *not* the cached web data, so startup only has
# to rescan the filesystem
ncacli rm -f /var/local/news/nca/cache/finder.cache

resetfakemount
nca_compose start web workers
