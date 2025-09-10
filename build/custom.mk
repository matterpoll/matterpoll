# Include custome targets and environment variables here

GO_TEST_FLAGS = -race -gcflags=-l
.DEFAULT_GOAL := all

## Generate mocks
.PHONY: mocks
mocks:
	cd server && go tool mockery
