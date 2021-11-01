// Code generated by 'fieldr'; DO NOT EDIT.

package as_map

type (
	EmbeddedAddressField    string
	EmbeddedAddressTag      string
	EmbeddedAddressTagValue string
)

const (
	embeddedAddressField_ZipCode     = EmbeddedAddressField("ZipCode")
	embeddedAddressField_AddressLine = EmbeddedAddressField("AddressLine")

	EmbeddedAddressTag_toMap = EmbeddedAddressTag("toMap")

	embeddedAddressTagValue_toMap_ZipCode     = EmbeddedAddressTagValue("zip_code")
	embeddedAddressTagValue_toMap_AddressLine = EmbeddedAddressTagValue("address_line")
)

func (v *EmbeddedAddress) AsMap() map[EmbeddedAddressField]interface{} {
	return map[EmbeddedAddressField]interface{}{
		embeddedAddressField_ZipCode:     v.ZipCode,
		embeddedAddressField_AddressLine: v.AddressLine,
	}
}

func (v *EmbeddedAddress) AsTagMap(tag EmbeddedAddressTag) map[EmbeddedAddressTagValue]interface{} {
	switch tag {
	case EmbeddedAddressTag_toMap:
		return map[EmbeddedAddressTagValue]interface{}{
			embeddedAddressTagValue_toMap_ZipCode:     v.ZipCode,
			embeddedAddressTagValue_toMap_AddressLine: v.AddressLine,
		}
	}
	return nil
}
