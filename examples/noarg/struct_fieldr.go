// Code generated by 'fieldr'; DO NOT EDIT.

package noarg

const (
	structFieldID                 = "ID"
	structFieldName               = "Name"
	structFieldSurname            = "Surname"
	structFieldNoExport           = "noExport"
	structFieldNoTag              = "NoTag"
	structFieldIgnoredInTagMap    = "IgnoredInTagMap"
	structFieldAddressZipCode     = "Address.ZipCode"
	structFieldAddressAddressLine = "Address.AddressLine"

	structTagToMap = "toMap"

	structTagValueToMapID                 = "id"
	structTagValueToMapName               = "name"
	structTagValueToMapSurname            = "surname"
	structTagValueToMapNoExport           = "no_export"
	structTagValueToMapNoTag              = "NoTag" //empty tag
	structTagValueToMapAddressZipCode     = "zip_code"
	structTagValueToMapAddressAddressLine = "address_line"
)

var (
	structFields = []string{
		structFieldID,
		structFieldName,
		structFieldSurname,
		structFieldNoTag,
		structFieldIgnoredInTagMap,
		structFieldAddressZipCode,
		structFieldAddressAddressLine,
	}

	structTags = []string{structTagToMap}

	structFieldTags = map[string][]string{
		structFieldID:                 []string{structTagToMap},
		structFieldName:               []string{structTagToMap},
		structFieldSurname:            []string{structTagToMap},
		structFieldNoTag:              []string{structTagToMap},
		structFieldIgnoredInTagMap:    []string{},
		structFieldAddressZipCode:     []string{structTagToMap},
		structFieldAddressAddressLine: []string{structTagToMap},
	}

	structTagValuesToMap = []string{
		structTagValueToMapID,
		structTagValueToMapName,
		structTagValueToMapSurname,
		structTagValueToMapNoTag,
		structTagValueToMapAddressZipCode,
		structTagValueToMapAddressAddressLine,
	}

	structTagValues = map[string][]string{
		structTagToMap: []string{
			structTagValueToMapID,
			structTagValueToMapName,
			structTagValueToMapSurname,
			structTagValueToMapNoTag,
			structTagValueToMapAddressZipCode,
			structTagValueToMapAddressAddressLine,
		},
	}

	structTagFields = map[string][]string{
		structTagToMap: []string{
			structFieldID,
			structFieldName,
			structFieldSurname,
			structFieldNoTag,
			structFieldAddressZipCode,
			structFieldAddressAddressLine,
		},
	}

	structFieldTagValue = map[string]map[string]string{
		structFieldID:                 map[string]string{structTagToMap: structTagValueToMapID},
		structFieldName:               map[string]string{structTagToMap: structTagValueToMapName},
		structFieldSurname:            map[string]string{structTagToMap: structTagValueToMapSurname},
		structFieldNoTag:              map[string]string{structTagToMap: structTagValueToMapNoTag},
		structFieldIgnoredInTagMap:    map[string]string{},
		structFieldAddressZipCode:     map[string]string{structTagToMap: structTagValueToMapAddressZipCode},
		structFieldAddressAddressLine: map[string]string{structTagToMap: structTagValueToMapAddressAddressLine},
	}
)

func (v *Struct) getFieldValue(field string) interface{} {
	switch field {
	case structFieldID:
		return v.ID
	case structFieldName:
		return v.Name
	case structFieldSurname:
		return v.Surname
	case structFieldNoTag:
		return v.NoTag
	case structFieldIgnoredInTagMap:
		return v.IgnoredInTagMap
	case structFieldAddressZipCode:
		return v.Address.ZipCode
	case structFieldAddressAddressLine:
		return v.Address.AddressLine
	}
	return nil
}

func (v *Struct) getFieldValueByTagValue(tag string) interface{} {
	switch tag {
	case structTagValueToMapID:
		return v.ID
	case structTagValueToMapName:
		return v.Name
	case structTagValueToMapSurname:
		return v.Surname
	case structTagValueToMapNoTag:
		return v.NoTag
	case structTagValueToMapAddressZipCode:
		return v.Address.ZipCode
	case structTagValueToMapAddressAddressLine:
		return v.Address.AddressLine
	}
	return nil
}

func (v *Struct) getFieldValuesByTag(tag string) []interface{} {
	switch tag {
	case structTagToMap:
		return []interface{}{
			v.ID,
			v.Name,
			v.Surname,
			v.NoTag,
			v.Address.ZipCode,
			v.Address.AddressLine,
		}
	}
	return nil
}

func (v *Struct) getFieldValuesByTagToMap() []interface{} {
	return []interface{}{
		v.ID,
		v.Name,
		v.Surname,
		v.NoTag,
		v.Address.ZipCode,
		v.Address.AddressLine,
	}
}

func (v *Struct) asMap() map[string]interface{} {
	return map[string]interface{}{
		structFieldID:                 v.ID,
		structFieldName:               v.Name,
		structFieldSurname:            v.Surname,
		structFieldNoTag:              v.NoTag,
		structFieldIgnoredInTagMap:    v.IgnoredInTagMap,
		structFieldAddressZipCode:     v.Address.ZipCode,
		structFieldAddressAddressLine: v.Address.AddressLine,
	}
}

func (v *Struct) asTagMap(tag string) map[string]interface{} {
	switch tag {
	case structTagToMap:
		return map[string]interface{}{
			structTagValueToMapID:                 v.ID,
			structTagValueToMapName:               v.Name,
			structTagValueToMapSurname:            v.Surname,
			structTagValueToMapNoTag:              v.NoTag,
			structTagValueToMapAddressZipCode:     v.Address.ZipCode,
			structTagValueToMapAddressAddressLine: v.Address.AddressLine,
		}
	}
	return nil
}
