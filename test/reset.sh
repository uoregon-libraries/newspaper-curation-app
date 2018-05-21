#!/bin/bash
set -eu

pushd .
cd ..

# Fry the issues, jobs, and batches from the database
docker-compose exec db mysql -unca -pnca -Dnca -e "delete from jobs; delete from issues; delete from job_logs; delete from batches;"

# Remove the finder cache, but *not* the cached web data - this gives us a
# quicker upstart since we'll only have to rescan the filesystem
docker-compose exec workers rm /var/local/news/nca/cache/finder.cache

docker-compose down

popd
./makemine.sh
rm fakemount/* -rf
go run copy-sources.go .
./make-older.sh

pushd .
cd ..
docker-compose up -d
popd
