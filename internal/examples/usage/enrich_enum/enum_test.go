package enrich_enum

import (
	"testing"

	"github.com/m4gshm/gollections/slice"
	"github.com/stretchr/testify/assert"
)

func Test_Enum(t *testing.T) {
	values := EnumValues()

	assert.Equal(t, []Enum{AA, BB, CC, DD}, values)

	names := slice.Convert(values, Enum.String)

	assert.Equal(t, []string{"AA", "BB", "CC", "DD"}, names)
}
