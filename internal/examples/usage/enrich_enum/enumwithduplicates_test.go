package enrich_enum

import (
	"testing"

	"github.com/m4gshm/gollections/slice"
	"github.com/stretchr/testify/assert"
)

func Test_EnumWithDuplicatesValues(t *testing.T) {
	values := EnumWithDuplicatesValues()

	assert.Equal(t, []EnumWithDuplicates{A, B, C}, values)
	assert.Equal(t, []EnumWithDuplicates{A, F, C}, values)

	names := slice.Convert(values, EnumWithDuplicates.String)

	assert.Equal(t, [][]string{{"A"}, {"B", "F"}, {"C"}}, names)
}
