#!/usr/bin/env bash
set -eu

make
cat $1 | \
  tidy -xml -i -w 999999 --sort-attributes alpha 2>/dev/null | \
  sed "s|<?xml version='.*|<?xml version=\"1.0\" encoding=\"utf-8\"?>|" > /tmp/old.xml

./bin/rewrite-mets -c ./settings -i /tmp/old.xml -o /tmp/xml
cat /tmp/xml | tidy -xml -i --sort-attributes alpha -w 999999 > /tmp/new.xml 2>/dev/null
IFS=''
diff=$(diff -uw /tmp/old.xml /tmp/new.xml  | grep "^[-+]" || true)

if [[ $diff == "" ]]; then
  echo "No errors!"
else
  echo "Check the diff:"
  echo
  echo $diff
fi
