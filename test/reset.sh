#!/bin/bash
./makemine.sh
rm fakemount/* -rf
go run copy-sources.go .
