package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderEmpty(t *testing.T) {
	actual := EntityBuilder[int32]{}.Build()
	assert.Equal(t, &Entity[int32]{BaseEntity: &BaseEntity[int32]{}}, actual)
}

func TestBuilderFields(t *testing.T) {
	actual := EntityBuilder[int32]{ID: 1, Name: "2"}.Build()
	assert.Equal(t, &Entity[int32]{BaseEntity: &BaseEntity[int32]{ID: 1}, Name: "2"}, actual)
}
