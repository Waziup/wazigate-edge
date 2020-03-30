#!/bin/bash

# export GO111MODULE="on"
# go mod init; go mod tidy

BIN="wazigate-edge"

go build -o $BIN .
