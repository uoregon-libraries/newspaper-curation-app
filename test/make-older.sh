#!/bin/bash
./makemine.sh
daysago=${1:-4}
if [[ -d fakemount ]]; then
  find fakemount/ -exec touch -d "$daysago days ago" {} \;
fi
