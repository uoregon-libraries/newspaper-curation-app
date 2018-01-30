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

echo Stopping service...
sudo systemctl stop blackmamba

echo Removing the old stuff
sudo rm -f /usr/local/black-mamba/server
sudo rm -f /usr/local/black-mamba/blackmamba.service
sudo rm /usr/local/black-mamba/static/ -rf
sudo rm /usr/local/black-mamba/templates/ -rf

echo Copying in the new stuff
src=$(pwd)
sudo cp $src/bin/server /usr/local/black-mamba/server
sudo cp $src/rhel7/blackmamba.service /usr/local/black-mamba/
sudo cp -r $src/templates/ /usr/local/black-mamba/
sudo cp -r $src/static/ /usr/local/black-mamba/

echo Doing a daemon reload and starting the service
sudo systemctl daemon-reload
sudo systemctl start blackmamba

git checkout master
