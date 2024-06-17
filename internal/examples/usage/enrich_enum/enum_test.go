package enrich_enum

import (
	"testing"

	"github.com/m4gshm/gollections/slice"
	"github.com/stretchr/testify/assert"
)

func Test_Enum(t *testing.T) {
	var (
		expectedValues = []Enum{AA, BB, CC, DD}
		values         = EnumAll()
		strings        = []string{"AA", "BB", "CC", "DD"}
		ints           = []int{1, 2, 3, 4}
	)

	assert.Equal(t, expectedValues, values)

	names := slice.Convert(values, Enum.Name)

	assert.Equal(t, strings, names)

	assert.Equal(t, expectedValues, slice.ConvertOK(strings, EnumByName))
	assert.Equal(t, expectedValues, slice.ConvertOK(ints, EnumByValue))
}
