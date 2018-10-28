# Ensure that go is installed. Note that this is independent of whether or not a server is being
# built, since the build script itself uses go.
ifeq ($(GO),)
    $(error "go is not available: see https://golang.org/doc/install")
endif

# Ensure that the build tools are compiled. Go's caching makes this quick.
$(shell cd build/manifest && $(GO) build -o ../bin/manifest)

# Extract the plugin id from the manifest.
PLUGIN_ID ?= $(shell build/bin/manifest id)
ifeq ($(PLUGIN_ID),)
    $(error "Cannot parse id from $(MANIFEST_FILE)")
endif

# Extract the plugin version from the manifest.
PLUGIN_VERSION ?= $(shell build/bin/manifest version)
ifeq ($(PLUGIN_VERSION),)
    $(error "Cannot parse version from $(MANIFEST_FILE)")
endif

# Determine if a server is defined in the manifest.
HAS_SERVER ?= $(shell build/bin/manifest has_server)

# Determine if a webapp is defined in the manifest.
HAS_WEBAPP ?= $(shell build/bin/manifest has_webapp)

# Try looking for dep in $(GOPATH) in case $(GOPATH)/bin isn't in $(PATH).
GOPATH ?= $(shell $(GO) env GOPATH)
ifeq ($(DEP),)
ifneq ($(wildcard $(GOPATH)/bin/dep),)
    DEP = $(GOPATH)/bin/dep
endif
endif

# Ensure that dep is installed.
ifneq ($(HAS_SERVER),)
ifeq ($(DEP),)
    $(error "dep is not available: see https://golang.github.io/dep/docs/installation.html")
endif
endif

# Ensure that npm (and thus node) is installed.
ifneq ($(HAS_WEBAPP),)
ifeq ($(NPM),)
    $(error "npm is not available: see https://www.npmjs.com/get-npm")
endif
endif
