#!/bin/sh

# export GO111MODULE="on"
# go mod init; go mod tidy

# Build wazigate-dashboard
cd wazigate-dashboard
npm i && npm run build
cd ..

# Build wazigate(-edge)
go build -ldflags "-s -w" -o wazigate .
