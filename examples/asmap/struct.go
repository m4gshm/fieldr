package asmap

import "time"

//go:generate fieldr -type EmbeddedAddress -out address_as_map.go -wrap -export -AsMap -AsTagMap
//go:generate fieldr -type Struct -out struct_as_map.go -wrap -export -AsMap -transform type:EmbeddedAddress:fmt=%v.AsMap() -flat FlatPrefix -flat FlatNoPrefix
//go:generate fieldr -type Struct -out struct_as_map.go -wrap -export -AsTagMap -transform fmt=&%v -transform type:EmbeddedAddress:fmt=%v.AsTagMap(EmbeddedAddressTag(tag))

type BaseStruct struct {
	ID int       `toMap:"id"`
	TS time.Time `toMap:"ts"`
}

type EmbeddedAddress struct {
	ZipCode     int    `toMap:"zip_code"`
	AddressLine string `toMap:"address_line"`
}

type FlatPart struct {
	CardNum string `toMap:"card_num"`
	Bank    string `toMap:"bank"`
}

type Struct struct {
	BaseStruct
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
	Address         *EmbeddedAddress `toMap:"address"`
	FlatNoPrefix    FlatPart         `toMap:""`
	FlatPrefix      FlatPart         `toMap:"flat"`
}
