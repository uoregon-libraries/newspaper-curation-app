#!/bin/bash
docker-compose exec db mysql -unca -pnca -Dnca -e "delete from jobs; delete from issues; delete from job_logs; delete from batches;"
docker-compose down
./makemine.sh
rm fakemount/* -rf
go run copy-sources.go .
./make-older.sh
docker-compose up -d
