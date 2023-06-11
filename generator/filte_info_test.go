package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_TypeReceiverVar(t *testing.T) {

	result := TypeReceiverVar("1asd")

	assert.Equal(t, "a", result)

	emptyResult := TypeReceiverVar("")

	assert.Equal(t, "r", emptyResult)

	splitResult := TypeReceiverVar("qw.erty")

	assert.Equal(t, "e", splitResult)

	splitEmpty := TypeReceiverVar("qw.") //???

	assert.Equal(t, "r", splitEmpty)
}
