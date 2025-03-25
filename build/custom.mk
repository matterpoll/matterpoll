# Include custome targets and environment variables here

GO_TEST_FLAGS = -race -gcflags=-l
.DEFAULT_GOAL := all

## Generate mocks
.PHONY: mocks
mocks:
	$(GO) install github.com/vektra/mockery/v2/...@v2.53.3
	cd  server && mockery
