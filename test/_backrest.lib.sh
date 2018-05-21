#!/bin/bash
set -eu

dcdown() {
  pushd .
  cd ..
  docker-compose down
  popd
}

dcup() {
  pushd .
  cd ..
  docker-compose up -d
  popd
}

dobackup() {
  rm -f ./nca.tgz
  rm -f ./fakemount.tar

  dcdown

  pushd .
  cd /var/lib/docker/volumes/nca_db/_data
  tar -czf ./nca.tgz nca
  popd

  mv /var/lib/docker/volumes/nca_db/_data/nca.tgz ./nca.tgz
  tar -cf ./fakemount.tar ./fakemount

  dcup
}

dorestore() {
  dcdown

  cp ./nca.tgz /var/lib/docker/volumes/nca_db/_data

  pushd .
  cd /var/lib/docker/volumes/nca_db/_data
  rm nca -rf
  tar -xzf ./nca.tgz
  popd

  rm fakemount -rf
  tar -xf ./fakemount.tar

  dcup
}
