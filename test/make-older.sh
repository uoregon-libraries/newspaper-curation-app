#!/bin/bash
./makemine.sh
if [[ -d fakemount ]]; then
  find fakemount/ -exec touch -d "4 days ago" {} \;
fi
