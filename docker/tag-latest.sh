#!/usr/bin/env bash
set -eu

docker-compose build app
tag=$(git tag | grep "^v[0-9.]\+$" | sort | tail -1)
dname=uolibraries/nca_app
docker rmi $dname:$tag || true
docker tag $dname $dname:$tag
