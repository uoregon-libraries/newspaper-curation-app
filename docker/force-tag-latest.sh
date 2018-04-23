#!/usr/bin/env bash
set -eu

echo Removing all nca_app images
docker rmi $(docker images | grep nca_app | awk '{print $1 ":" $2}') || true
./docker/tag-latest.sh
