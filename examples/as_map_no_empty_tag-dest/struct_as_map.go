package as_map_no_empty_tag_dest

import (
	as_map "fmt"
	"unsafe"

	as_map1 "example/asmap"
)

type (
	StructField     string
	StructFieldList []StructField
	StructTag       string
	StructTagValue  string
)

const (
	StructFieldID              = StructField("ID")
	StructFieldTS              = StructField("TS")
	StructFieldName            = StructField("Name")
	StructFieldSurname         = StructField("Surname")
	structFieldNoExport        = StructField("noExport")
	StructFieldNoTag           = StructField("NoTag")
	StructFieldIgnoredInTagMap = StructField("IgnoredInTagMap")
	StructFieldAddress         = StructField("Address")
	StructFieldFlatNoPrefix    = StructField("FlatNoPrefix")
	StructFieldFlatPrefix      = StructField("FlatPrefix")

	StructTagToMap = StructTag("toMap")

	StructTagValueToMapID         = StructTagValue("id")
	StructTagValueToMapTS         = StructTagValue("ts")
	StructTagValueToMapName       = StructTagValue("name")
	StructTagValueToMapSurname    = StructTagValue("surname")
	structTagValueToMapNoExport   = StructTagValue("no_export")
	StructTagValueToMapAddress    = StructTagValue("address")
	StructTagValueToMapFlatPrefix = StructTagValue("flat")
)

var (
	structFields = StructFieldList{
		StructFieldID,
		StructFieldTS,
		StructFieldName,
		StructFieldSurname,
		StructFieldNoTag,
		StructFieldIgnoredInTagMap,
		StructFieldAddress,
		StructFieldFlatNoPrefix,
		StructFieldFlatPrefix,
	}
)

func init() {
	as_map.Print("just for fun")
}

func AsMap(v *as_map1.Struct) map[StructField]interface{} {
	return map[StructField]interface{}{
		StructFieldID:              v.ID,
		StructFieldTS:              v.TS,
		StructFieldName:            v.Name,
		StructFieldSurname:         v.Surname,
		StructFieldNoTag:           v.NoTag,
		StructFieldIgnoredInTagMap: v.IgnoredInTagMap,
		StructFieldAddress:         v.Address,
		StructFieldFlatNoPrefix:    v.FlatNoPrefix,
		StructFieldFlatPrefix:      v.FlatPrefix,
	}
}

func AsTagMap(v *as_map1.Struct, tag StructTag) map[StructTagValue]interface{} {
	switch tag {
	case StructTagToMap:
		return map[StructTagValue]interface{}{
			StructTagValueToMapID:         v.ID,
			StructTagValueToMapTS:         v.TS,
			StructTagValueToMapName:       v.Name,
			StructTagValueToMapSurname:    v.Surname,
			StructTagValueToMapAddress:    v.Address,
			StructTagValueToMapFlatPrefix: v.FlatPrefix,
		}
	}
	return nil
}

func (v StructFieldList) Strings() []string {
	return *(*[]string)(unsafe.Pointer(&v))
}
