#!/bin/bash

# This should be considered a working example... but not necessarily the best
# way to deploy this to production!  Tweak for your own environment.

version=$(git tag | tail -1)
echo "Checking out $version and recompiling for deployment"
git co $version
make clean
make

echo Stopping service...
sudo systemctl stop p2cgo

echo Removing the old stuff
sudo rm -f /usr/local/p2c-go/p2cgo.service
sudo rm /usr/local/p2c-go/static/ -rf
sudo rm /usr/local/p2c-go/templates/ -rf

echo Copying in the new stuff
src=$(pwd)
sudo cp $src/bin/server /usr/local/p2c-go/server
sudo cp $src/p2cgo.service /usr/local/p2c-go/
sudo cp -r $src/templates/ /usr/local/p2c-go/
sudo cp -r $src/static/ /usr/local/p2c-go/

echo Doing a daemon reload and starting the service
sudo systemctl daemon-reload
sudo systemctl start p2cgo
