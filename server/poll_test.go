package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	p1 := &Poll{Question: "Question",
		Options: []*Option{
			{Answer: "Answer 1"},
			{Answer: "Answer 2"},
		},
	}
	p2 := Decode(p1.Encode())
	assert.Equal(t, p1, p2)
}
