#!/bin/bash

REPO="github.com/Waziup/wazigate-edge"
BIN="wazigate-edge"
SERVICE="wazigate-edge"

go install $REPO

systemctl stop $SERVICE
systemctl disable $SERVICE.service

cp "$GOPATH/bin/$BIN" "/bin/$BIN"
cp "$GOPATH/src/$REPO/$SERVICE.service" /lib/systemd/system/$SERVICE.service

systemctl enable $SERVICE.service
