#!/bin/bash

# This should be considered a working example... but not necessarily the best
# way to deploy this to production!  Tweak for your own environment.

version=$(git tag | tail -1)
echo "Checking out $version and recompiling for deployment"
status=$(git status --porcelain | grep -v "^??")
if [[ $status != "" ]]; then
  echo "Stash changes to deploy"
  exit 1
fi
git checkout $version
make clean
make

echo Stopping services...
sudo systemctl stop httpd
sudo systemctl stop blackmamba-httpd
sudo systemctl stop blackmamba-workers

echo Removing the old stuff
sudo rm -f /usr/local/black-mamba/server
sudo rm -f /usr/local/black-mamba/run-jobs
sudo rm -f /usr/local/black-mamba/blackmamba-httpd.service
sudo rm -f /usr/local/black-mamba/blackmamba-workers.service
sudo rm /usr/local/black-mamba/static/ -rf
sudo rm /usr/local/black-mamba/templates/ -rf

echo Removing the cache
sudo rm /tmp/black-mamba/finder.cache -f

echo Migrating the database
goose --env production up

echo Copying in the new stuff
src=$(pwd)
dst="/usr/local/black-mamba"
sudo cp $src/bin/server $dst/
sudo cp $src/bin/run-jobs $dst/
sudo cp $src/rhel7/blackmamba-httpd.service $dst/
sudo cp $src/rhel7/blackmamba-workers.service $dst/
sudo cp -r $src/templates/ $dst/
sudo cp -r $src/static/ $dst/

echo Doing a daemon reload and starting the service
sudo systemctl daemon-reload
sudo systemctl start blackmamba-workers
sudo systemctl start blackmamba-httpd
sudo systemctl start httpd

git checkout master
