package stringify_enum

import (
	"testing"

	"github.com/m4gshm/gollections/slice"
	"github.com/stretchr/testify/assert"
)

func Test_Enum(t *testing.T) {
	var (
		expectedValues = []Enum{AA, BB, CC, DD}
		values         = EnumValues()
		strings        = []string{"AA", "BB", "CC", "DD"}
	)

	assert.Equal(t, expectedValues, values)

	names := slice.Convert(values, Enum.String)

	assert.Equal(t, strings, names)

	assert.Equal(t, expectedValues, slice.ConvertOK(strings, EnumFromString))
}
