.PHONY: build

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-w -s" -o ./oxcross-leaf

install: build
	cp ./oxcross-leaf /usr/local/bin/oxcross-leaf
	cp ./oxcross-leaf.service /etc/systemd/system/oxcross-leaf.service
	chmod 644 /etc/systemd/system/oxcross-leaf.service
