package multiply_tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UseDeepRefAccess(t *testing.T) {
	e := &Entity{}

	upAt := e.val(ENTITY_COL_UPDATED_AT)
	assert.Nil(t, upAt)

	upAt2 := e.val(ENTITY_COL_UPDATED_AT2)
	assert.Nil(t, upAt2)

	upAt3 := e.val(ENTITY_COL_UPDATED_AT3)
	assert.Nil(t, upAt3)

	// ue := new(UpdateableEntity)
	// uer1 := UpdateableEntityRef(ue)
	uer2 := new(UpdateableEntityRef) //&uer1
	uer3 := &uer2
	e.Upd = &uer3

	upAt = e.val(ENTITY_COL_UPDATED_AT)
	assert.Nil(t, upAt)
}
