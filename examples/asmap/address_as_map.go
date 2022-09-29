package asmap

type EmbeddedAddressField string

const (
	ZipCode     EmbeddedAddressField = "ZipCode"
	AddressLine EmbeddedAddressField = "AddressLine"
)

func (v *EmbeddedAddress) AsMap() map[EmbeddedAddressField]interface{} {
	if v == nil {
		return nil
	}
	m := map[EmbeddedAddressField]interface{}{}
	m[ZipCode] = v.ZipCode
	m[AddressLine] = v.AddressLine
	return m
}
