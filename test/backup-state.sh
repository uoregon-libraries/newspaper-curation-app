#!/bin/bash
set -eu

rm -f ./nca.tgz
rm -f ./fakemount.tar

docker-compose stop db
docker-compose rm -f db

pushd .
cd /var/lib/docker/volumes/nca_db/_data
tar -czf ./nca.tgz nca
popd
mv /var/lib/docker/volumes/nca_db/_data/nca.tgz ./nca.tgz
tar -cf ./fakemount.tar ./fakemount

docker-compose up -d db
