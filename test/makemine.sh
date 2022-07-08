#!/bin/bash
if [[ -d ./fakemount ]]; then
  iam=$(whoami)
  sudo chown -R $iam ./fakemount
  for dir in $(find ./fakemount -type d); do
    sudo chmod 755 $dir || true
  done
fi
