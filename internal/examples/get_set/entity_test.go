package get_set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_EmbeddedGetSet(t *testing.T) {
	entity := Entity[int]{BaseEntity: &BaseEntity[int]{RefCodeAwareEntity: &RefCodeAwareEntity{CodeAwareEntity: &CodeAwareEntity{}}}}

	code := "code"
	entity.SetCode(code)

	assert.Equal(t, code, entity.GetCode())
	assert.Equal(t, code, entity.BaseEntity.Code)
	assert.Equal(t, code, entity.Code)
}

func Test_EmbeddedGetSetNotInitialized(t *testing.T) {
	entity := Entity[int]{}
	entity.SetCode("code")
	assert.Equal(t, "", entity.GetCode())
}

func Test_EmbeddedGetSetNilObject(t *testing.T) {
	var entity *Entity[int]
	entity.SetCode("code")
	assert.Equal(t, "", entity.GetCode())
}
