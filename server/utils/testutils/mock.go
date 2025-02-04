package testutils

import (
	"github.com/stretchr/testify/mock"
)

func GetMockArgumentsWithType(typeString string, num int) []interface{} {
	ret := make([]interface{}, num)
	for i := 0; i < len(ret); i++ {
		ret[i] = mock.AnythingOfType(typeString)
	}
	return ret
}
