package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilderEmpty(t *testing.T) {
	actual := (&EntityBuilder[int32]{}).Build()
	expected := &Entity[int32]{BaseEntity: &BaseEntity[int32]{CodeAwareEntity: &CodeAwareEntity{}}}
	assert.Equal(t, expected, actual)
}

func TestBuilderFields(t *testing.T) {
	actual := *((&EntityBuilder[int32]{Name: "1"}).SetID(2).SetSurname("3").SetEmbedded(EmbeddedEntityBuilder{}.SetMetadata("meta").Build()).Build())
	expected := Entity[int32]{BaseEntity: &BaseEntity[int32]{ID: 2, CodeAwareEntity: &CodeAwareEntity{}}, Name: "1", Surname: "3", Embedded: EmbeddedEntity{Metadata: "meta"}}
	assert.Equal(t, expected, actual)
}

func TestToBuilder(t *testing.T) {
	builder := (&EntityBuilder[int32]{Name: "1"}).SetID(2).SetSurname("3").SetEmbedded(EmbeddedEntityBuilder{}.SetMetadata("meta").Build())
	object := builder.Build()
	restoredBuilder := object.ToBuilder()
	assert.Equal(t, builder, restoredBuilder)
}
