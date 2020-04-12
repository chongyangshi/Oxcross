#!/bin/sh

set -e

GO_VERSION=1.14.2

[ -f "/tmp/go$GO_VERSION.linux-amd64.tar.gz" ] || wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz -O /tmp/go1.14.2.linux-amd64.tar.gz
[ -d "/usr/local/go" ] || sudo tar -C /usr/local -zxf /tmp/go$GO_VERSION.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
rm -rf /tmp/go$GO_VERSION.linux-amd64.tar.gz

cd leaf
sudo make install
systemctl enable oxcross-leaf
systemctl start oxcross-leaf
systemctl status oxcross-leaf