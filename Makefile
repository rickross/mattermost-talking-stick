.PHONY: all build clean

PLUGIN_ID = com.gitschool.talking-stick
PLUGIN_VERSION = 0.2.6

all: build

build:
	cd server && go build -o dist/plugin-linux-amd64
	cd server && GOOS=linux GOARCH=arm64 go build -o dist/plugin-linux-arm64
	cd server && GOOS=darwin GOARCH=amd64 go build -o dist/plugin-darwin-amd64
	cd server && GOOS=darwin GOARCH=arm64 go build -o dist/plugin-darwin-arm64

dist: build
	mkdir -p dist
	tar -czf dist/$(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz \
		plugin.json \
		server/dist/plugin-* \
		assets/

clean:
	rm -rf server/dist
	rm -rf dist

deps:
	cd server && go mod download
	cd server && go mod tidy
