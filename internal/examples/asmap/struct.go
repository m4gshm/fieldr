package asmap

import "time"

//go:generate fieldr -type EmbeddedAddress -out address_as_map.go as-map -key-type . -export
//go:generate fieldr -type Struct -out struct_as_map.go as-map -key-type . -export -rewrite type:*EmbeddedAddress:fmt=%v.AsMap() -flat Flat

type BaseStruct struct {
	ID int
	TS *time.Time
}

type EmbeddedAddress struct {
	ZipCode     int
	AddressLine string
}

type FlatPart struct {
	CardNum string
	Bank    string
}

type Struct[n string] struct {
	*BaseStruct
	Name     n
	Surname  string
	noExport string //nolint
	NoTag    string
	Address  *EmbeddedAddress
	Flat     FlatPart
}
