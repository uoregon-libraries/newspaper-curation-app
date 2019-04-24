#!/bin/bash
iam=$(whoami)
sudo chown -R $iam ./fakemount
sudo chmod 755 ./fakemount/workflow/*
