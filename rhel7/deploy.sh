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
  checkout=$(git tag | grep "^v[0-9.]\+$" | sort -V | tail -1)
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

make clean
make

cp /tmp/old-version.go src/version/version.go

echo Stopping services...
sudo systemctl stop httpd || true
sudo systemctl stop nca-httpd || true
sudo systemctl stop nca-workers || true

src=$(pwd)
ncadir=/usr/local/nca
if [[ -d $ncadir ]]; then
  echo Removing the old stuff
  tmpdir=/tmp/nca-$(date +"%s")
  sudo mv $ncadir $tmpdir
  sudo mkdir $ncadir
  sudo mv $tmpdir/settings* $ncadir/
  sudo find $ncadir/ -mindepth 1 -maxdepth 1 -type f -not -name "settings*" -exec rm -f {} \;
  sudo rm $ncadir/templates/ -rf
  sudo rm $ncadir/static/ -rf
  sudo rm -f /etc/rsyslog.d/nca.conf
else
  echo First-time install detected: edit $ncadir/settings
  sudo mkdir $ncadir
  sudo cp $src/settings-example $ncadir/settings
fi

echo Removing the cache
sudo rm /tmp/nca/finder.cache -f

echo Migrating the database
goose --env production up

echo Copying in the new stuff
sudo cp $src/bin/server $ncadir/
sudo cp $src/bin/run-jobs $ncadir/
sudo cp $src/bin/batch-fixer $ncadir/
sudo cp $src/bin/queue-batches $ncadir/
sudo cp $src/bin/bulk-issue-queue $ncadir/
sudo cp $src/bin/move-errored-issues $ncadir/
sudo cp $src/bin/delete-live-done-issues $ncadir/
sudo cp $src/rhel7/nca-httpd.service $ncadir/
sudo cp $src/rhel7/nca-workers.service $ncadir/
sudo cp $src/rhel7/nca-rsyslog.conf /etc/rsyslog.d/nca.conf
sudo cp -r $src/templates/ $ncadir/
sudo cp -r $src/static/ $ncadir/

echo Doing a daemon reload and starting the service
sudo systemctl enable $ncadir/nca-httpd.service
sudo systemctl enable $ncadir/nca-workers.service
sudo systemctl daemon-reload
sudo systemctl start nca-workers
sudo systemctl start nca-httpd
sudo systemctl restart rsyslog

sudo systemctl start httpd
sudo rm -rf $tmpdir
