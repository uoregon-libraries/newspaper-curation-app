#!/usr/bin/env bash
set -eu

curref=$(git b | grep "^\*" | sed "s|^\* ||")
tag=$(git tag | grep "^v[0-9.]\+$" | sort | tail -1)
dname=uolibraries/nca_app

git co $tag
docker-compose build app
docker rmi $dname:$tag || true
docker tag $dname $dname:$tag

git co $curref
