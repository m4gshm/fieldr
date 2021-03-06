// Code generated by 'fieldr'; DO NOT EDIT.

package json

const (
	StructFieldID       = "ID"
	StructFieldName     = "Name"
	StructFieldSurname  = "Surname"
	StructFieldNoJson   = "NoJson"
	structFieldNoExport = "noExport"
	StructFieldNoTag    = "NoTag"

	StructTagJson = "json"

	StructTagValueJsonID       = "id"
	StructTagValueJsonName     = "name,omitempty"
	StructTagValueJsonSurname  = "surname,omitempty"
	StructTagValueJsonNoJson   = "-"
	structTagValueJsonNoExport = "no_export"
)

var (
	structFields = []string{
		StructFieldID,
		StructFieldName,
		StructFieldSurname,
		StructFieldNoJson,
		StructFieldNoTag,
	}

	structFieldTagValue = map[string]map[string]string{
		StructFieldID:      map[string]string{StructTagJson: StructTagValueJsonID},
		StructFieldName:    map[string]string{StructTagJson: StructTagValueJsonName},
		StructFieldSurname: map[string]string{StructTagJson: StructTagValueJsonSurname},
		StructFieldNoJson:  map[string]string{StructTagJson: StructTagValueJsonNoJson},
		StructFieldNoTag:   map[string]string{},
	}
)

func (v *Struct) GetFieldValue(field string) interface{} {
	switch field {
	case StructFieldID:
		return v.ID
	case StructFieldName:
		return v.Name
	case StructFieldSurname:
		return v.Surname
	case StructFieldNoJson:
		return v.NoJson
	case StructFieldNoTag:
		return v.NoTag
	}
	return nil
}
