.PHONY: build
GO_VERSION := 1.14.2

build:
	[ -f "/tmp/go1.14.2.linux-amd64.tar.gz" ] || wget https://dl.google.com/go/go$(GO_VERSION).linux-amd64.tar.gz -O /tmp/go1.14.2.linux-amd64.tar.gz
	[ -d "/usr/local/go" ] || tar -C /usr/local -zxf /tmp/go$(GO_VERSION).linux-amd64.tar.gz
	GOPATH=$(cd $(CURDIR)/.. && pwd)/vendor GOOS=linux GOARCH=amd64 CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-w -s" -o ./oxcross-origin
	rm -rf /tmp/go$(GO_VERSION).linux-amd64.tar.gz

install: build
	cp ./oxcross-origin /usr/local/bin/oxcross-origin
	cp ./oxcross-origin.service /etc/systemd/system/oxcross-origin.service
	chmod 644 /etc/systemd/system/oxcross-origin.service
