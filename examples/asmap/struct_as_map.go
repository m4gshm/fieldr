// Code generated by 'fieldr'; DO NOT EDIT.

package asmap

type StructField string

const (
	StructFieldID                  StructField = "ID"
	StructFieldTS                  StructField = "TS"
	StructFieldName                StructField = "Name"
	StructFieldSurname             StructField = "Surname"
	structFieldNoExport            StructField = "noExport"
	StructFieldNoTag               StructField = "NoTag"
	StructFieldIgnoredInTagMap     StructField = "IgnoredInTagMap"
	StructFieldAddress             StructField = "Address"
	StructFieldFlatNoPrefixCardNum StructField = "FlatNoPrefix.CardNum"
	StructFieldFlatNoPrefixBank    StructField = "FlatNoPrefix.Bank"
	StructFieldFlatPrefixCardNum   StructField = "FlatPrefix.CardNum"
	StructFieldFlatPrefixBank      StructField = "FlatPrefix.Bank"
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
