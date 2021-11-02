package as_map

type (
	Tag            string
	StructField    string
	StructTag      string
	StructTagValue string
)

const (
	structField_ID              = StructField("ID")
	structField_TS              = StructField("TS")
	structField_Name            = StructField("Name")
	structField_Surname         = StructField("Surname")
	StructField_noExport        = StructField("noExport")
	structField_NoTag           = StructField("NoTag")
	structField_IgnoredInTagMap = StructField("IgnoredInTagMap")
	structField_Address         = StructField("Address")

	StructTag_toMap = StructTag("toMap")

	structTagValue_toMap_ID       = StructTagValue("id")
	structTagValue_toMap_TS       = StructTagValue("ts")
	structTagValue_toMap_Name     = StructTagValue("name")
	structTagValue_toMap_Surname  = StructTagValue("surname")
	StructTagValue_toMap_noExport = StructTagValue("no_export")
	structTagValue_toMap_NoTag    = StructTagValue("NoTag") //empty tag
	structTagValue_toMap_Address  = StructTagValue("address")
)

func (v *Struct) AsMap() map[StructField]interface{} {
	return map[StructField]interface{}{
		structField_ID:              v.ID,
		structField_TS:              v.TS,
		structField_Name:            v.Name,
		structField_Surname:         v.Surname,
		structField_NoTag:           v.NoTag,
		structField_IgnoredInTagMap: v.IgnoredInTagMap,
		structField_Address:         v.Address.AsMap(),
	}
}

func (v *Struct) AsTagMap(tag StructTag) map[StructTagValue]interface{} {
	switch tag {
	case StructTag_toMap:
		return map[StructTagValue]interface{}{
			structTagValue_toMap_ID:      &v.ID,
			structTagValue_toMap_TS:      &v.TS,
			structTagValue_toMap_Name:    &v.Name,
			structTagValue_toMap_Surname: &v.Surname,
			structTagValue_toMap_NoTag:   &v.NoTag,
			structTagValue_toMap_Address: v.Address.AsTagMap(EmbeddedAddressTag(tag)),
		}
	}
	return nil
}
