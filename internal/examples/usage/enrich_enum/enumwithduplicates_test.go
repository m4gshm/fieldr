package enrich_enum

import (
	"testing"

	"github.com/m4gshm/gollections/slice"
	"github.com/stretchr/testify/assert"
)

func Test_EnumWithDuplicatesValues(t *testing.T) {
	values := EnumWithDuplicatesAll()

	assert.Equal(t, slice.Of(A, B, C), values)
	assert.Equal(t, slice.Of(A, F, C), values)

	names := slice.Convert(values, EnumWithDuplicates.Name)

	assert.Equal(t, [][]string{{"A"}, {"B", "F"}, {"C"}}, names)

	assert.Equal(t, slice.Of(A, B, F, C), slice.ConvertOK(slice.Of("A", "B", "F", "C"), EnumWithDuplicatesByName))
}
