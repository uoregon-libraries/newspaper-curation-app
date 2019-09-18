#!/bin/bash
if [[ -d ./fakemount ]]; then
  iam=$(whoami)
  sudo chown -R $iam ./fakemount
  sudo chmod 755 ./fakemount/workflow/* || true
fi
