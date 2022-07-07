#!/bin/bash
./makemine.sh
daysago=${1:-4}
if [[ -d fakemount ]]; then
  for file in $(find fakemount/ -type f -name ".manifest"); do
    cat $file | sed 's|"Created":"[^"]\+",|"Created":"1900-01-01T00:00:00.000000000-07:00",|' > $file-2
    mv $file-2 $file
  done
fi
