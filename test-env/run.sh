#!/bin/bash

## This builds Dispatch and deploys it on 2 test nodes
cd node
go build ../../dispatchd
go build ../../dispatchctl

cd ..
docker-compose up --build