#!/bin/bash

iam=$(whoami)
sudo chown -R $iam ./fakemount

lastdir=""
for file in $(find fakemount/page-review/ -name "*.pdf"); do
  dir=${file%/*}
  if [[ $dir != $lastdir ]]; then
    c=0
    lastdir=$dir
  fi
  let c=c+1
  newfile=$(printf "%04d.pdf" $c)
  echo "mv $file $dir/$newfile"
done
