#!/bin/bash

set -e

echo "updating dependencies"
go get -u
go mod tidy

echo "building bot binary"
go build

echo "installing bot"
mkdir -p "$HOME/bin"
mv "dotmtxbot" "$HOME/bin/dotmtxbot"

echo "installing systemd configuration"
mkdir -p "$HOME/.config/systemd/user"
cp -a "config/dotmtxbot.service" "$HOME/.config/systemd/user"

echo "enabling and restarting systemd service"
systemctl --user daemon-reload
systemctl --user enable dotmtxbot.service
systemctl --user restart dotmtxbot.service
