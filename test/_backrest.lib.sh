#!/bin/bash
set -eu

dcdown() {
  pushd .
  cd ..
  docker-compose down
  popd
}

getloc() {
  dt=$(date +"%Y-%m-%d_%H%M%S")
  loc="$(pwd)/backups/${1:-$dt}"
  echo $loc
}

dobackup() {
  loc=$(getloc "$1")

  if [[ -d $loc ]]; then
    echo "$loc already exists; aborting backup"
    return 1
  fi

  echo "Backing up to $loc in 2 seconds"
  sleep 2
  mkdir -p $loc
  dcdown

  pushd .
  cd /var/lib/docker/volumes/nca_db/_data
  tar -czf $loc/nca.tgz nca
  popd

  tar -cf $loc/fakemount.tar ./fakemount
}

dorestore() {
  loc=$(getloc "$1")
  echo $loc

  if [[ ! -d $loc ]]; then
    echo "$loc doesn't exist; aborting restore"
    return 1
  fi

  echo "Restoring from $loc in 2 seconds"
  sleep 2
  dcdown

  pushd .
  cd /var/lib/docker/volumes/nca_db/_data
  rm nca -rf
  tar -xzf $loc/nca.tgz
  popd

  rm fakemount -rf
  tar -xf $loc/fakemount.tar
}
