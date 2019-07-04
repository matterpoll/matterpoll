GO ?= $(shell command -v go 2> /dev/null)
NPM ?= $(shell command -v npm 2> /dev/null)
CURL ?= $(shell command -v curl 2> /dev/null)
MANIFEST_FILE ?= plugin.json
GO_TEST_FLAGS ?= -race
GO_BUILD_FLAGS ?=
MM_UTILITIES_DIR ?= ../mattermost-utilities

export GO111MODULE=on

# You can include assets this directory into the bundle. This can be e.g. used to include profile pictures.
ASSETS_DIR ?= assets

# Verify environment, and define PLUGIN_ID, PLUGIN_VERSION, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

# Include custom makefile, if pressent
ifneq ($(wildcard build/custom.mk),)
	include build/custom.mk
endif

## Checks the code style, tests, builds and bundles the plugin.
all: check-style test dist

## Propagates plugin manifest information into the server/ and webapp/ folders as required.
.PHONY: apply
apply:
	./build/bin/manifest apply

## Runs govet and gofmt against all packages.
.PHONY: check-style
check-style: webapp/.npminstall gofmt govet golint
	@echo Checking for style guide compliance

ifneq ($(HAS_WEBAPP),)
	cd webapp && npm run lint
endif

## Runs gofmt against all packages.
.PHONY: gofmt
gofmt:
ifneq ($(HAS_SERVER),)
	@echo Running gofmt
	@for package in $$(go list ./...); do \
		echo "Checking "$$package; \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				echo "Gofmt failure"; \
				exit 1; \
			fi; \
		fi; \
	done
	@echo Gofmt success
endif

## Runs govet against all packages.
.PHONY: govet
govet:
ifneq ($(HAS_SERVER),)
	@echo Running govet
	@# Workaroung because you can't install binaries without adding them to go.mod 
	env GO111MODULE=off $(GO) get golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
	$(GO) vet ./...
	$(GO) vet -vettool=$(GOPATH)/bin/shadow ./...
	@echo Govet success
endif

## Runs golint against all packages.
.PHONY: golint
golint:
	@echo Running lint
	env GO111MODULE=off $(GO) get golang.org/x/lint/golint
	golint -set_exit_status ./...
	@echo lint success

## Builds the server, if it exists, including support for multiple architectures.
.PHONY: server
server:
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist;
	cd server && env GOOS=linux GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -o dist/plugin-linux-amd64;
	cd server && env GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -o dist/plugin-darwin-amd64;
	cd server && env GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -o dist/plugin-windows-amd64.exe;
endif

## Ensures NPM dependencies are installed without having to run this all the time.
webapp/.npminstall:
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

## Builds the webapp, if it exists.
.PHONY: webapp
webapp: webapp/.npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run build;
endif

## Generates a tar bundle of the plugin for install.
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
ifneq ($(wildcard $(ASSETS_DIR)/.),)
	cp -r $(ASSETS_DIR) dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_PUBLIC),)
	cp -r public/ dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server/dist;
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/;
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_ID)/webapp/dist;
	cp -r webapp/dist/* dist/$(PLUGIN_ID)/webapp/dist/;
endif
	cd dist && tar -cvzf $(BUNDLE_NAME) $(PLUGIN_ID)

	@echo plugin built at: dist/$(BUNDLE_NAME)

## Builds and bundles the plugin.
.PHONY: dist
dist:	apply server webapp bundle

## Installs the plugin to a (development) server.
.PHONY: deploy
deploy: dist
## It uses the API if appropriate environment variables are defined,
## or copying the files directly to a sibling mattermost-server directory.
ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	@echo "Installing plugin via API"
	$(eval TOKEN := $(shell curl -i --post301 --location $(MM_SERVICESETTINGS_SITEURL) -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep Token | cut -f2 -d' ' 2> /dev/null))
	@curl -s --post301 --location $(MM_SERVICESETTINGS_SITEURL) -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins -F "plugin=@dist/$(BUNDLE_NAME)" -F "force=true" > /dev/null && \
		curl -s --post301 --location $(MM_SERVICESETTINGS_SITEURL) -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null && \
		echo "OK." || echo "Sorry, something went wrong."
else ifneq ($(wildcard ../mattermost-server/.*),)
	@echo "Installing plugin via filesystem. Server restart and manual plugin enabling required"
	mkdir -p ../mattermost-server/plugins
	tar -C ../mattermost-server/plugins -zxvf dist/$(BUNDLE_NAME)
else
	@echo "No supported deployment method available. Install plugin manually."
endif

## Runs any lints and unit tests defined for the server and webapp, if they exist.
.PHONY: test
test: webapp/.npminstall
ifneq ($(HAS_SERVER),)
	$(GO) test -v $(GO_TEST_FLAGS) ./server/...
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run fix;
endif

## Creates a coverage report for the server code.
.PHONY: coverage
coverage: webapp/.npminstall
ifneq ($(HAS_SERVER),)
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=server/coverage.txt ./server/...
	$(GO) tool cover -html=server/coverage.txt
endif

## Extract strings for translation from the source code.
.PHONY: i18n-extract
i18n-extract:
ifneq ($(HAS_WEBAPP),)
ifeq ($(HAS_MM_UTILITIES),)
	@echo "You must clone github.com/mattermost/mattermost-utilities repo in .. to use this command"
else
	cd $(MM_UTILITIES_DIR) && npm install && npm run babel && node mmjstool/build/index.js i18n extract-webapp --webapp-dir $(PWD)/webapp
endif
endif

## Clean removes all build artifacts.
.PHONY: clean
clean:
	rm -fr dist/
ifneq ($(HAS_SERVER),)
	rm -fr server/dist
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr webapp/.npminstall
	rm -fr webapp/dist
	rm -fr webapp/node_modules
endif
	rm -fr build/bin/

# Help documentatin à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@cat Makefile | grep -v '\.PHONY' |  grep -v '\help:' | grep -B1 -E '^[a-zA-Z0-9_.-]+:.*' | sed -e "s/:.*//" | sed -e "s/^## //" |  grep -v '\-\-' | sed '1!G;h;$$!d' | awk 'NR%2{printf "\033[36m%-30s\033[0m",$$0;next;}1' | sort
