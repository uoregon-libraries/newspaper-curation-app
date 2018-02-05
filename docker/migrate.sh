#!/usr/bin/env bash
wait_for_db() {
  MAX_TRIES=15
  TRIES=0
  while true; do
    echo "Attempting to connect to database..."
    mysql -uroot -hdb -p123456 -e 'ALTER DATABASE blackmamba charset=utf8'
    local st=$?
    if [[ $st == 0 ]]; then
      return
    fi
    echo "DB not responding yet; waiting 5 seconds"
    let TRIES++
    sleep 5
    if [ "$TRIES" = "$MAX_TRIES" ]; then
      echo "ERROR: Unable to connect to the database"
      exit 2
    fi
  done
}
 
cd /usr/local/black-mamba
wait_for_db
goose up
