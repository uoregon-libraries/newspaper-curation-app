#!/usr/bin/env bash
set -eu

curref=$(git b | grep "^\*" | sed "s|^\* ||")
tag=$(git describe --abbrev=0)
dname=uolibraries/nca_app

git co $tag
docker-compose build app
docker rmi $dname:$tag || true
docker tag $dname $dname:$tag

git co $curref
