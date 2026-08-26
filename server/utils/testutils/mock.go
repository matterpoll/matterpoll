package testutils

import (
	"github.com/stretchr/testify/mock"
)

func GetMockArgumentsWithType(typeString string, num int) []any {
	ret := make([]any, num)
	for i := range ret {
		ret[i] = mock.AnythingOfType(typeString)
	}
	return ret
}
