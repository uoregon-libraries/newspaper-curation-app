#!/bin/bash
iam=$(whoami)
sudo chown -R $iam ./fakemount
find fakemount/ -exec touch -d "4 days ago" {} \;
