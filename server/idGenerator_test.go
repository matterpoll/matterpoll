package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPollIDGenerator(t *testing.T) {
	assert := assert.New(t)
	idGen := PollIDGenerator{}
	numberOfRuns := 100

	ids := make([]string, 0)

	for index := 0; index < numberOfRuns; index++ {
		id := idGen.NewId()
		assert.Len(id, 26)
		for i := range ids {
			assert.NotEqual(i, id)
		}
		ids = append(ids, id)
	}
}
