package as_map

import "time"

//go:generate fieldr -type EmbeddedAddress -out address_as_map.go -wrap -export -AsMap -AsTagMap
//go:generate fieldr -type Struct -out struct_as_map.go -wrap -export -AsMap -transform type:EmbeddedAddress:fmt=%v.AsMap()
//go:generate fieldr -type Struct -out struct_as_map.go -wrap -export -AsTagMap -transform :fmt=&%v -transform type:EmbeddedAddress:fmt=%v.AsTagMap(EmbeddedAddressTag(tag))

type BaseStruct struct {
	ID int       `toMap:"id"`
	TS time.Time `toMap:"ts"`
}

type EmbeddedAddress struct {
	ZipCode     int    `toMap:"zip_code"`
	AddressLine string `toMap:"address_line"`
}

type Struct struct {
	BaseStruct
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
	Address         EmbeddedAddress `toMap:"address"`
}
