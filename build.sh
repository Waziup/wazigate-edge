#!/bin/sh

# export GO111MODULE="on"
# go mod init; go mod tidy

# Build wazigate-dashboard
cd wazigate-dashboard
npm i --force && npm run build --force
cd ..

# Build wazigate(-edge)
export GOARCH=arm64 GOOS=linux
go build -ldflags "-s -w" -o wazigate .
