package asmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructAsMapEmpty(t *testing.T) {
	s := Struct{}
	m := s.AsMap()
	assert.Equal(t, map[StructField]interface{}{
		Name:      "",
		Surname:   "",
		"NoTag":   "",
		"CardNum": "",
		"Bank":    "",
	}, m)
}

func TestStructAsMapEmptyEmbeddedRef(t *testing.T) {
	s := Struct{BaseStruct: &BaseStruct{}, Name: "N", Address: &EmbeddedAddress{}}
	m := s.AsMap()
	assert.Equal(t, map[StructField]interface{}{
		BaseStructID: 0,
		Name:         "N",
		Surname:      "",
		"NoTag":      "",
		"CardNum":    "",
		"Bank":       "",
		Address: map[EmbeddedAddressField]interface{}{
			AddressLine: "",
			ZipCode:     0,
		},
	}, m)

}
