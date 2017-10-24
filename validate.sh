#!/usr/bin/bash
#
# validate.sh runs various code analysis tools to reduce the likelihood of
# unnoticed errors
set -eu

IFS=''
unformatted=$(find src/ -name "*.go" | xargs gofmt -l -s)
linter=$(golint src/...)
vet=$(go vet -printfuncs Debugf,Infof,Warnf,Errorf,Criticalf,Fatalf ./src/...  2>&1 || true)

result=0

if [[ $unformatted != "" ]]; then
  echo "gofmt reports issues:"
  echo "---------------------"
  echo $unformatted
  result=1
fi

if [[ $linter != "" ]]; then
  if [[ $result == 1 ]]; then
    echo
  fi
  echo "linter reports issues:"
  echo "----------------------"
  echo $linter
  result=1
fi

if [[ $vet != "" ]]; then
  if [[ $result == 1 ]]; then
    echo
  fi
  echo "go vet reports issues:"
  echo "----------------------"
  echo $vet
  result=1
fi

if [[ $result == 1 ]]; then
  echo
fi
exit $result
