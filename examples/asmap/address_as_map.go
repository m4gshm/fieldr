package asmap

type EmbeddedAddressField string

const (
	ZipCode     EmbeddedAddressField = "ZipCode"
	AddressLine EmbeddedAddressField = "AddressLine"
)

func (e *EmbeddedAddress) AsMap() map[EmbeddedAddressField]interface{} {
	if e == nil {
		return nil
	}
	m := map[EmbeddedAddressField]interface{}{}
	m[ZipCode] = e.ZipCode
	m[AddressLine] = e.AddressLine
	return m
}
