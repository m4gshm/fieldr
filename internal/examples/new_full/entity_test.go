package new_full

import (
	"example/sql_base"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_EmbeddedGetSet(t *testing.T) {
	code := "code"

	entity := NewEntity(
		1,
		code,
		2,
		struct {
			Schema  string
			Version int
		}{Schema: "schema", Version: 123},
		NoDBFieldsEntity{},
		"name",
		"surname",
		[]int32{1, 2, 3},
		make([]*time.Time, 10, 100),
		sql_base.VersionedEntity{},
		nil,
		nil,
		EmbeddedEntity{},
		nil,
	)

	assert.Equal(t, code, entity.E.CodeAwareEntity.Code)
	assert.Equal(t, code, entity.E.Code)
	assert.Equal(t, code, entity.Code)

	assert.Equal(t, 123, entity.metadata.Version)

	entity2 := *New2(
		&E[int]{
			ID: 1,
			RefCodeAwareEntity: &RefCodeAwareEntity{
				CodeAwareEntity: &CodeAwareEntity{
					Code: code,
				},
			},
			foreignIDAwareEntity: foreignIDAwareEntity[int]{
				ForeignID: 2,
			},
		},
		struct {
			Schema  string
			Version int
		}{Schema: "schema", Version: 123},
		NoDBFieldsEntity{},
		"name",
		"surname",
		[]int32{1, 2, 3},
		make([]*time.Time, 10, 100),
		sql_base.VersionedEntity{},
		nil,
		nil,
		EmbeddedEntity{},
		nil,
	)

	assert.Equal(t, entity, entity2)

}
