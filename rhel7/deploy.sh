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
ncadir=/usr/local/nca
tmpdir=/tmp/nca-$(date +"%s")
sudo mv $ncadir $tmpdir
sudo mkdir $ncadir
sudo mv $tmpdir/settings* $ncadir/
sudo find $ncadir/ -mindepth 1 -maxdepth 1 -type f -not -name "settings*" -exec rm -f {} \;
sudo rm $ncadir/templates/ -rf
sudo rm $ncadir/static/ -rf

echo Removing the cache
sudo rm /tmp/nca/finder.cache -f

echo Migrating the database
goose --env production up

echo Copying in the new stuff
src=$(pwd)
sudo cp $src/bin/server $ncadir/
sudo cp $src/bin/run-jobs $ncadir/
sudo cp $src/bin/queue-batches $ncadir/
sudo cp $src/bin/bulk-issue-queue $ncadir/
sudo cp $src/rhel7/nca-httpd.service $ncadir/
sudo cp $src/rhel7/nca-workers.service $ncadir/
sudo cp -r $src/templates/ $ncadir/
sudo cp -r $src/static/ $ncadir/

echo Doing a daemon reload and starting the service
sudo systemctl daemon-reload
sudo systemctl start nca-workers
sudo systemctl start nca-httpd

echo Waiting 30 seconds for NCA to finish scanning issues
sleep 30
sudo systemctl start httpd
sudo rm -rf $tmpdir
