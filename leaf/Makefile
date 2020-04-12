.PHONY: build
GO_VERSION := 1.14.2

build:
	[ ! -f "/tmp/go1.14.2.linux-amd64.tar.gz" ] || wget https://dl.google.com/go/go$(GO_VERSION).linux-amd64.tar.gz -O /tmp/go1.14.2.linux-amd64.tar.gz
	[ ! -f "/usr/local/go" ] || tar -C /usr/local -zxf /tmp/go$(GO_VERSION).linux-amd64.tar.gz
	export PATH=$PATH:/usr/local/go/bin
	go get -d -v
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./oxcross-leaf
	rm -rf /tmp/go$(GO_VERSION).linux-amd64.tar.gz

install: build
	cp ./oxcross-leaf /usr/local/bin/oxcross-leaf
	cp ./oxcross-leaf.service /etc/systemd/system/oxcross-leaf.service
	chmod 644 /etc/systemd/system/oxcross-leaf.service
