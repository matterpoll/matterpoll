# Include custome targets and environment variables here

GO_TEST_FLAGS = -race -gcflags=-l
.DEFAULT_GOAL := all

## Generate store mocks
.PHONY: store-mocks
store-mocks:
	mockery -name=".Store" -dir server/store -output server/store/mockstore/mocks -note 'Regenerate this file using `make store-mocks`.'
