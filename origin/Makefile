.PHONY: build

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-w -s" -mod vendor -o ./oxcross-origin

install: build
	cp ./oxcross-origin /usr/local/bin/oxcross-origin
	cp ./oxcross-origin.service /etc/systemd/system/oxcross-origin.service
	chmod 644 /etc/systemd/system/oxcross-origin.service
