.PHONY: build
GO_VERSION := 1.14.2

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-w -s" -o ./oxcross-leaf
	rm -rf /tmp/go$(GO_VERSION).linux-amd64.tar.gz

install: build
	cp ./oxcross-leaf /usr/local/bin/oxcross-leaf
	cp ./oxcross-leaf.service /etc/systemd/system/oxcross-leaf.service
	chmod 644 /etc/systemd/system/oxcross-leaf.service
