// Code generated by 'fieldr -type Struct -export -out struct_util.go -Fields -FieldTagValueMap -GetFieldValue -EnumTags -EnumTagValues'; DO NOT EDIT.

package json

const (
	StructField_ID       = "ID"
	StructField_Name     = "Name"
	StructField_Surname  = "Surname"
	StructField_NoJson   = "NoJson"
	structField_noExport = "noExport"
	StructField_NoTag    = "NoTag"

	StructTag_json = "json"

	StructTagValue_json_ID       = "id"
	StructTagValue_json_Name     = "name,omitempty"
	StructTagValue_json_Surname  = "surname,omitempty"
	StructTagValue_json_NoJson   = "-"
	structTagValue_json_noExport = "no_export"
)

var (
	struct_Fields = []string{
		StructField_ID,
		StructField_Name,
		StructField_Surname,
		StructField_NoJson,
		StructField_NoTag,
	}

	struct_TagValues_json = []string{
		StructTagValue_json_ID,
		StructTagValue_json_Name,
		StructTagValue_json_Surname,
		StructTagValue_json_NoJson,
	}

	struct_FieldTagValue = map[string]map[string]string{
		StructField_ID:      map[string]string{StructTag_json: StructTagValue_json_ID},
		StructField_Name:    map[string]string{StructTag_json: StructTagValue_json_Name},
		StructField_Surname: map[string]string{StructTag_json: StructTagValue_json_Surname},
		StructField_NoJson:  map[string]string{StructTag_json: StructTagValue_json_NoJson},
		StructField_NoTag:   map[string]string{},
	}
)

func (v *Struct) GetFieldValue(field string) interface{} {
	switch field {
	case StructField_ID:
		return v.ID
	case StructField_Name:
		return v.Name
	case StructField_Surname:
		return v.Surname
	case StructField_NoJson:
		return v.NoJson
	case StructField_NoTag:
		return v.NoTag
	}
	return nil
}
