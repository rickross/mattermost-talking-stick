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
	rm -rf dist/stage
	mkdir -p dist/stage/$(PLUGIN_ID)/server/dist
	mkdir -p dist/stage/$(PLUGIN_ID)/assets
	# Server
	cp plugin.json dist/stage/$(PLUGIN_ID)/
	cp server/dist/plugin-* dist/stage/$(PLUGIN_ID)/server/dist/
	# Webapp (optional, but included when present)
	if [ -f webapp/dist/main.js ]; then \
		mkdir -p dist/stage/$(PLUGIN_ID)/webapp/dist; \
		cp webapp/dist/main.js dist/stage/$(PLUGIN_ID)/webapp/dist/; \
		if [ -f webapp/dist/main.js.LICENSE.txt ]; then \
			cp webapp/dist/main.js.LICENSE.txt dist/stage/$(PLUGIN_ID)/webapp/dist/; \
		fi; \
	fi
	# Create bundle with top-level plugin directory
	mkdir -p dist
	tar -czf dist/$(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz -C dist/stage $(PLUGIN_ID)

clean:
	rm -rf server/dist
	rm -rf dist

deps:
	cd server && go mod download
	cd server && go mod tidy
