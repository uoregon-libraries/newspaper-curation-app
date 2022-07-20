#!/bin/bash
./makemine.sh
daysago=${1:-4}
dt=$(date -d "$daysago days ago" +"%Y-%m-%dT00:00:00.000000000-07:00")

if [[ -d fakemount ]]; then
  for file in $(find fakemount/ -type f -name ".manifest"); do
    sed -i 's|"Created":"[^"]\+",|"Created":"'$dt'",|' $file
  done
fi
