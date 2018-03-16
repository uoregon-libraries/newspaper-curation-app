#!/bin/bash
set -eu

docker-compose stop db
docker-compose rm -f db

cp ./nca.tgz /var/lib/docker/volumes/nca_db/_data
pushd .
cd /var/lib/docker/volumes/nca_db/_data
rm nca -rf
tar -xzf ./nca.tgz
popd

rm fakemount -rf
tar -xf ./fakemount.tar

docker-compose up -d db
