package new_opts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EmbeddedGetSet(t *testing.T) {
	code := "code"

	entity := NewEntity(
		WithCode[int](code),
		WithMetadata[int](struct {
			Schema  string
			Version int
		}{Schema: "schema", Version: 123}),
	)

	assert.Equal(t, code, entity.E.CodeAwareEntity.Code)
	assert.Equal(t, code, entity.E.Code)
	assert.Equal(t, code, entity.Code)

	assert.Equal(t, 123, entity.metadata.Version)

}
