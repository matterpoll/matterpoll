GO ?= $(shell command -v go 2> /dev/null)
DEP ?= $(shell command -v dep 2> /dev/null)
NPM ?= $(shell command -v npm 2> /dev/null)
HTTP ?= $(shell command -v http 2> /dev/null)
CURL ?= $(shell command -v curl 2> /dev/null)
MANIFEST_FILE ?= plugin.json

# Verify environment, and define PLUGIN_ID, HAS_SERVER and HAS_WEBAPP as needed.
include build/setup.mk

# all, the default target, tests, builds and bundles the plugin.
all: test dist

# apply propagates the plugin id into the server/ and webapp/ folders as required.
.PHONY: apply
apply:
	./build/bin/manifest apply

# server/.depensure ensures the server dependencies are installed
server/.depensure:
ifneq ($(HAS_SERVER),)
	cd server && $(DEP) ensure
	touch $@
endif

# server builds the server, if it exists, including support for multiple architectures
.PHONY: server
server: server/.depensure
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist;
	cd server && env GOOS=linux GOARCH=amd64 $(GO) build -o dist/plugin-linux-amd64;
	cd server && env GOOS=darwin GOARCH=amd64 $(GO) build -o dist/plugin-darwin-amd64;
	cd server && env GOOS=windows GOARCH=amd64 $(GO) build -o dist/plugin-windows-amd64.exe;
endif

# webapp/.npminstall ensures NPM dependencies are installed without having to run this all the time
webapp/.npminstall:
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

# webapp builds the webapp, if it exists
.PHONY: webapp
webapp: webapp/.npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run build;
endif

# bundle generates a tar bundle of the plugin for install
.PHONY: bundle
bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server/dist;
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/;
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_ID)/webapp/dist;
	cp -r webapp/dist/* dist/$(PLUGIN_ID)/webapp/dist/;
endif
	cd dist && tar -cvzf $(PLUGIN_ID).tar.gz $(PLUGIN_ID)

	@echo plugin built at: dist/$(PLUGIN_ID).tar.gz

# dist builds and bundles the plugin
.PHONY: dist
dist: apply \
      server \
      webapp \
      bundle

# deploy installs the plugin to a (development) server, using the API if appropriate environment
# variables are defined, or copying the files directly to a sibling mattermost-server directory
.PHONY: deploy
deploy: dist
ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(HTTP)),)
	@echo "Installing plugin via API"
		(TOKEN=`http --print h POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login login_id=$(MM_ADMIN_USERNAME) password=$(MM_ADMIN_PASSWORD) | grep Token | cut -f2 -d' '` && \
		  http --print b GET $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/me Authorization:"Bearer $$TOKEN" && \
			http --print b DELETE $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID) Authorization:"Bearer $$TOKEN" && \
			http --print b --check-status --form POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins plugin@dist/$(PLUGIN_ID).tar.gz Authorization:"Bearer $$TOKEN" && \
		  http --print b POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable Authorization:"Bearer $$TOKEN" && \
		  http --print b POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/logout Authorization:"Bearer $$TOKEN" \
	  )
else ifneq ($(and $(MM_SERVICESETTINGS_SITEURL),$(MM_ADMIN_USERNAME),$(MM_ADMIN_PASSWORD),$(CURL)),)
	@echo "Installing plugin via API"
	$(eval TOKEN := $(shell curl -i -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/users/login -d '{"login_id": "$(MM_ADMIN_USERNAME)", "password": "$(MM_ADMIN_PASSWORD)"}' | grep Token | cut -f2 -d' ' 2> /dev/null))
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X DELETE $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID) > /dev/null
	@curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins -F "plugin=@dist/$(PLUGIN_ID).tar.gz" > /dev/null && \
		curl -s -H "Authorization: Bearer $(TOKEN)" -X POST $(MM_SERVICESETTINGS_SITEURL)/api/v4/plugins/$(PLUGIN_ID)/enable > /dev/null && \
		echo "OK." || echo "Sorry, something went wrong."
else ifneq ($(wildcard ../mattermost-server/.*),)
	@echo "Installing plugin via filesystem. Server restart and manual plugin enabling required"
	mkdir -p ../mattermost-server/plugins
	tar -C ../mattermost-server/plugins -zxvf dist/$(PLUGIN_ID).tar.gz
else
	@echo "No supported deployment method available. Install plugin manually."
endif

# test runs any lints and unit tests defined for the server and webapp, if they exist
.PHONY: test
test: server/.depensure webapp/.npminstall
ifneq ($(HAS_SERVER),)
	cd server && $(GO) test -v -coverprofile=coverage.txt ./...
endif
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run fix;
endif

# clean removes all build artifacts
.PHONY: clean
clean:
	rm -fr dist/
ifneq ($(HAS_SERVER),)
	rm -fr server/dist
	rm -fr server/.depensure
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr webapp/.npminstall
	rm -fr webapp/dist
	rm -fr webapp/node_modules
endif
	rm -fr build/bin/
