#!/bin/bash
set -eu

sudo su -c "source ./_backrest.lib.sh && dorestore"
