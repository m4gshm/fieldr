package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderEmpty(t *testing.T) {
	actual := EntityBuilder{}.Build()
	assert.Equal(t, &Entity{BaseEntity: &BaseEntity{}}, actual)
}

func TestBuilderFields(t *testing.T) {
	actual := EntityBuilder{ID: 1, Name: "2"}.Build()
	assert.Equal(t, &Entity{BaseEntity: &BaseEntity{ID: 1}, Name: "2"}, actual)
}
