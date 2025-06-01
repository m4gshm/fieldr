package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NexExpr(t *testing.T) {
	res := generateNewObjectExpr("s", "**SomeType[ID]")
	assert.Equal(t,
		`s1 := new(SomeType[ID])
s0 := &s1
s = &s0`, res)
}
