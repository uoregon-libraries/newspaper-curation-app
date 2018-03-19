#!/bin/bash

# This should be considered a working example... but not necessarily the best
# way to deploy this to production!  Tweak for your own environment.

set -eu

type=${1:-}

case "$type" in

"dev")
  checkout=
  version="-$(git log -1 --format="%h")"
  ;;

"prod")
  checkout=$(git tag | grep -v ".-rc" | tail -1)
  version=
  ;;

*)
  echo "You must specify 'dev' or 'prod'"
  exit 1
esac

set +e
status=$(git status --porcelain | grep -v "^??")
set -e
if [[ $status != "" ]]; then
  echo "Stash changes to deploy"
  exit 1
fi


if [[ $checkout != "" ]]; then
  git checkout $checkout
fi

cp src/version/version.go /tmp/old-version.go
sed -i "s|\"$|$version\"|" src/version/version.go

make cleanall
make

cp /tmp/old-version.go src/version/version.go

echo Stopping services...
sudo systemctl stop httpd
sudo systemctl stop nca-httpd
sudo systemctl stop nca-workers

echo Removing the old stuff
sudo rm -f /usr/local/nca/server
sudo rm -f /usr/local/nca/run-jobs
sudo rm -f /usr/local/nca/nca-httpd.service
sudo rm -f /usr/local/nca/nca-workers.service
sudo rm /usr/local/nca/static/ -rf
sudo rm /usr/local/nca/templates/ -rf

echo Removing the cache
sudo rm /tmp/nca/finder.cache -f

echo Migrating the database
goose --env production up

echo Copying in the new stuff
src=$(pwd)
dst="/usr/local/nca"
sudo cp $src/bin/server $dst/
sudo cp $src/bin/run-jobs $dst/
sudo cp $src/rhel7/nca-httpd.service $dst/
sudo cp $src/rhel7/nca-workers.service $dst/
sudo cp -r $src/templates/ $dst/
sudo cp -r $src/static/ $dst/

echo Doing a daemon reload and starting the service
sudo systemctl daemon-reload
sudo systemctl start nca-workers
sudo systemctl start nca-httpd
sudo systemctl start httpd
