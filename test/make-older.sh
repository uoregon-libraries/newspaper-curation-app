#!/bin/bash
./makemine.sh
daysago=${1:-4}
dt=$(date -d "$daysago days ago" +"%Y-%m-%dT00:00:00.000000000-07:00")

if [[ -d fakemount ]]; then
  for file in $(find fakemount/ -type f -name ".manifest"); do
    cat $file | sed 's|"Created":"[^"]\+",|"Created":"'$dt'",|' > $file-2
    mv $file-2 $file
  done
fi
