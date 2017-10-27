#!/usr/bin/bash
#
# dev-server.sh fires up a server listening on port 12346 and assuming a suffix
# of "-indev" for Apache routing.  When Go source or template files change, it
# auto-kills the server, recompiles, and starts the server back up.  Requires a
# working settings file in the current directory.
set -eu

port=${1:-12346}
suffix=${2:--indev}
debugflag="--debug"

if [[ ! -z ${NODEBUG:-} ]]; then
  debugflag=""
fi

oldmd5=""
pid=""

killserver() {
  if [[ $pid != "" ]]; then
    while true; do
      echo "Attempting to terminate the server"
      ps -p $pid 2>/dev/null >/dev/null || break
      kill $pid
      ps -p $pid 2>/dev/null >/dev/null || break
      sleep 1
    done
  fi
  pid=""
}

trap "killserver" SIGINT

while true; do
  srcs_md5=$(find src/ -type f -name "*.go" | xargs cat | md5sum)
  tmpl_md5=$(find templates/ -type f -name "*.html" | xargs cat | md5sum)
  if [[ $oldmd5 != "$srcs_md5$tmpl_md5" ]]; then
    killserver

    echo "Rebuilding..."

    last_build_md5=""
    while true; do
      oldifs=$IFS
      IFS=''
      gb build >/dev/null 2>.build.txt && break || true
      build_md5=$(md5sum .build.txt)
      if [[ $last_build_md5 != "$build_md5" ]]; then
        last_build_md5=$build_md5
        echo
        cat .build.txt
      fi
      IFS=$oldifs
      sleep 5
    done

    # In case we were in a build loop for a while and lots of things changed,
    # recalculate the checksums and store them
    srcs_md5=$(find src/ -type f -name "*.go" | xargs cat | md5sum)
    tmpl_md5=$(find templates/ -type f -name "*.html" | xargs cat | md5sum)
    oldmd5="$srcs_md5$tmpl_md5"

    ./bin/server \
        -c ./settings \
        --chronam-web-root http://oregonnews.uoregon.edu \
        -p $port \
        --webroot=/odnp-admin/sftpreport$suffix \
        --parent-webroot=/odnp-admin \
        --static-files $(pwd)/static \
        --cache-path $(pwd)/tmp \
        $debugflag \
        $(pwd)/templates &
      pid=$!
    fi
  sleep 1
done
