#!/bin/bash
pushd .
cd ..
docker-compose exec db mysql -unca -pnca -Dnca -e "delete from jobs; delete from issues; delete from job_logs; delete from batches;"
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
