package as_map_no_empty_tag_dest

import (
	asMap2 "example/as-map"
)

type (
	StructField    string
	StructTag      string
	StructTagValue string
)

const (
	StructField_ID              = StructField("ID")
	StructField_Name            = StructField("Name")
	StructField_Surname         = StructField("Surname")
	structField_noExport        = StructField("noExport") //nolint
	StructField_NoTag           = StructField("NoTag")
	StructField_IgnoredInTagMap = StructField("IgnoredInTagMap")

	StructTag_toMap = StructTag("toMap")

	StructTagValue_toMap_ID       = StructTagValue("id")
	StructTagValue_toMap_Name     = StructTagValue("name")
	StructTagValue_toMap_Surname  = StructTagValue("surname")
	structTagValue_toMap_noExport = StructTagValue("no_export") //nolint
)

func AsMap(v *asMap2.Struct) map[StructField]interface{} {
	return map[StructField]interface{}{
		StructField_ID:              v.ID,
		StructField_Name:            v.Name,
		StructField_Surname:         v.Surname,
		StructField_NoTag:           v.NoTag,
		StructField_IgnoredInTagMap: v.IgnoredInTagMap,
	}
}

func AsTagMap(v *asMap2.Struct, tag StructTag) map[StructTagValue]interface{} {
	switch tag {
	case StructTag_toMap:
		return map[StructTagValue]interface{}{
			StructTagValue_toMap_ID:      v.ID,
			StructTagValue_toMap_Name:    v.Name,
			StructTagValue_toMap_Surname: v.Surname,
		}
	}
	return nil
}
