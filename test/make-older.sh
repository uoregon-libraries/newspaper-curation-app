#!/bin/bash
./makemine.sh
find fakemount/ -exec touch -d "4 days ago" {} \;
