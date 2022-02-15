package asmap

type (
	StructField    string
	StructTag      string
	StructTagValue string
)

const (
	StructFieldID                  = StructField("ID")
	StructFieldTS                  = StructField("TS")
	StructFieldName                = StructField("Name")
	StructFieldSurname             = StructField("Surname")
	structFieldNoExport            = StructField("noExport") //nolint
	StructFieldNoTag               = StructField("NoTag")
	StructFieldIgnoredInTagMap     = StructField("IgnoredInTagMap")
	StructFieldAddress             = StructField("Address")
	StructFieldFlatNoPrefix        = StructField("FlatNoPrefix")
	StructFieldFlatNoPrefixCardNum = StructField("FlatNoPrefix.CardNum")
	StructFieldFlatNoPrefixBank    = StructField("FlatNoPrefix.Bank")
	StructFieldFlatPrefix          = StructField("FlatPrefix")
	StructFieldFlatPrefixCardNum   = StructField("FlatPrefix.CardNum")
	StructFieldFlatPrefixBank      = StructField("FlatPrefix.Bank")

	StructTagToMap = StructTag("toMap")

	StructTagValueToMapID           = StructTagValue("id")
	StructTagValueToMapTS           = StructTagValue("ts")
	StructTagValueToMapName         = StructTagValue("name")
	StructTagValueToMapSurname      = StructTagValue("surname")
	structTagValueToMapNoExport     = StructTagValue("no_export") //nolint
	StructTagValueToMapNoTag        = StructTagValue("NoTag")     //empty tag
	StructTagValueToMapAddress      = StructTagValue("address")
	StructTagValueToMapFlatNoPrefix = StructTagValue("FlatNoPrefix") //empty tag
	StructTagValueToMapFlatPrefix   = StructTagValue("flat")
)

func (v *Struct) AsMap() map[StructField]interface{} {
	return map[StructField]interface{}{
		StructFieldID:                  v.ID,
		StructFieldTS:                  v.TS,
		StructFieldName:                v.Name,
		StructFieldSurname:             v.Surname,
		StructFieldNoTag:               v.NoTag,
		StructFieldIgnoredInTagMap:     v.IgnoredInTagMap,
		StructFieldAddress:             v.Address.AsMap(),
		StructFieldFlatNoPrefixCardNum: v.FlatNoPrefix.CardNum,
		StructFieldFlatNoPrefixBank:    v.FlatNoPrefix.Bank,
		StructFieldFlatPrefixCardNum:   v.FlatPrefix.CardNum,
		StructFieldFlatPrefixBank:      v.FlatPrefix.Bank,
	}
}

func (v *Struct) AsTagMap(tag StructTag) map[StructTagValue]interface{} {
	switch tag {
	case StructTagToMap:
		return map[StructTagValue]interface{}{
			StructTagValueToMapID:           &v.ID,
			StructTagValueToMapTS:           &v.TS,
			StructTagValueToMapName:         &v.Name,
			StructTagValueToMapSurname:      &v.Surname,
			StructTagValueToMapNoTag:        &v.NoTag,
			StructTagValueToMapAddress:      v.Address.AsTagMap(EmbeddedAddressTag(tag)),
			StructTagValueToMapFlatNoPrefix: &v.FlatNoPrefix,
			StructTagValueToMapFlatPrefix:   &v.FlatPrefix,
		}
	}
	return nil
}
