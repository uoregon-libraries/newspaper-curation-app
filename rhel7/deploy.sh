#!/bin/bash

# This should be considered a working example... but not necessarily the best
# way to deploy this to production!  Tweak for your own environment.

set -eu

set +e
status=$(git status --porcelain | grep -v "^??")
set -e
if [[ $status != "" ]]; then
  echo "Stash changes to deploy"
  exit 1
fi

make clean
make

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
source $ncadir/settings
goose -dir $src/db/migrations mysql "$DB_USER:$DB_PASSWORD@tcp($DB_HOST:3306)/$DB_DATABASE" up

echo Copying in the new stuff
sudo cp $src/bin/server $ncadir/
sudo cp $src/bin/run-jobs $ncadir/
sudo cp $src/bin/batch-fixer $ncadir/
sudo cp $src/bin/queue-batches $ncadir/
sudo cp $src/bin/bulk-issue-queue $ncadir/
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
