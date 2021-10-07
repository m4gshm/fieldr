// Code generated by 'fieldr -type StructWrap -wrap -hardcode -compact'; DO NOT EDIT.

package noarg

type (
	structWrapField     string
	structWrapFields    []structWrapField
	structWrapTag       string
	structWrapTags      []structWrapTag
	structWrapTagValue  string
	structWrapTagValues []structWrapTagValue
)

const (
	structWrapField_ID              = structWrapField("ID")
	structWrapField_Name            = structWrapField("Name")
	structWrapField_Surname         = structWrapField("Surname")
	structWrapField_noExport        = structWrapField("noExport")
	structWrapField_NoTag           = structWrapField("NoTag")
	structWrapField_IgnoredInTagMap = structWrapField("IgnoredInTagMap")

	structWrapTag_toMap = structWrapTag("toMap")

	structWrapTagValue_toMap_ID       = structWrapTagValue("id")
	structWrapTagValue_toMap_Name     = structWrapTagValue("name")
	structWrapTagValue_toMap_Surname  = structWrapTagValue("surname")
	structWrapTagValue_toMap_noExport = structWrapTagValue("no_export")
	structWrapTagValue_toMap_NoTag    = structWrapTagValue("NoTag") //empty tag

)

func (v structWrapFields) strings() []string {
	strings := make([]string, len(v))
	for i, val := range v {
		strings[i] = string(val)
	}
	return strings
}

func (v structWrapTags) strings() []string {
	strings := make([]string, len(v))
	for i, val := range v {
		strings[i] = string(val)
	}
	return strings
}

func (v structWrapTagValues) strings() []string {
	strings := make([]string, len(v))
	for i, val := range v {
		strings[i] = string(val)
	}
	return strings
}

func (v structWrapFields) excludes(excludes ...structWrapField) structWrapFields {
	excl := make(map[structWrapField]interface{}, len(excludes))
	for _, e := range excludes {
		excl[e] = nil
	}
	withoutExcludes := make(structWrapFields, 0, len(v)-len(excludes))
	for _, _v := range v {
		if _, ok := excl[_v]; !ok {
			withoutExcludes = append(withoutExcludes, _v)
		}
	}
	return withoutExcludes
}

func (v structWrapTags) excludes(excludes ...structWrapTag) structWrapTags {
	excl := make(map[structWrapTag]interface{}, len(excludes))
	for _, e := range excludes {
		excl[e] = nil
	}
	withoutExcludes := make(structWrapTags, 0, len(v)-len(excludes))
	for _, _v := range v {
		if _, ok := excl[_v]; !ok {
			withoutExcludes = append(withoutExcludes, _v)
		}
	}
	return withoutExcludes
}

func (v structWrapTagValues) excludes(excludes ...structWrapTagValue) structWrapTagValues {
	excl := make(map[structWrapTagValue]interface{}, len(excludes))
	for _, e := range excludes {
		excl[e] = nil
	}
	withoutExcludes := make(structWrapTagValues, 0, len(v)-len(excludes))
	for _, _v := range v {
		if _, ok := excl[_v]; !ok {
			withoutExcludes = append(withoutExcludes, _v)
		}
	}
	return withoutExcludes
}

var (
	structWrap_Fields = structWrapFields{"ID", "Name", "Surname", "NoTag", "IgnoredInTagMap"}

	structWrap_Tags = structWrapTags{"toMap"}

	structWrap_FieldTags = map[structWrapField]structWrapTags{
		"ID":              structWrapTags{"toMap"},
		"Name":            structWrapTags{"toMap"},
		"Surname":         structWrapTags{"toMap"},
		"NoTag":           structWrapTags{"toMap"},
		"IgnoredInTagMap": structWrapTags{},
	}

	structWrap_TagValues_toMap = structWrapTagValues{"id", "name", "surname", ""}

	structWrap_TagValues = map[structWrapTag]structWrapTagValues{
		"toMap": structWrapTagValues{"id", "name", "surname", ""},
	}

	structWrap_TagFields = map[structWrapTag]structWrapFields{
		"toMap": structWrapFields{"ID", "Name", "Surname", "NoTag"},
	}

	structWrap_FieldTagValue = map[structWrapField]map[structWrapTag]structWrapTagValue{
		"ID":              map[structWrapTag]structWrapTagValue{"toMap": "id"},
		"Name":            map[structWrapTag]structWrapTagValue{"toMap": "name"},
		"Surname":         map[structWrapTag]structWrapTagValue{"toMap": "surname"},
		"NoTag":           map[structWrapTag]structWrapTagValue{"toMap": ""},
		"IgnoredInTagMap": map[structWrapTag]structWrapTagValue{},
	}
)

func (v *StructWrap) getFieldValue(field structWrapField) interface{} {
	switch field {
	case "ID":
		return v.ID
	case "Name":
		return v.Name
	case "Surname":
		return v.Surname
	case "NoTag":
		return v.NoTag
	case "IgnoredInTagMap":
		return v.IgnoredInTagMap
	}
	return nil
}

func (v *StructWrap) getFieldValueByTagValue(tag structWrapTagValue) interface{} {
	switch tag {
	case "id":
		return v.ID
	case "name":
		return v.Name
	case "surname":
		return v.Surname
	case "":
		return v.NoTag
	}
	return nil
}

func (v *StructWrap) getFieldValuesByTag(tag structWrapTag) []interface{} {
	switch tag {
	case "toMap":
		return []interface{}{v.ID, v.Name, v.Surname, v.NoTag}
	}
	return nil
}

func (v *StructWrap) asMap() map[structWrapField]interface{} {
	return map[structWrapField]interface{}{
		"ID":              v.ID,
		"Name":            v.Name,
		"Surname":         v.Surname,
		"NoTag":           v.NoTag,
		"IgnoredInTagMap": v.IgnoredInTagMap,
	}
}

func (v *StructWrap) asTagMap(tag structWrapTag) map[structWrapTagValue]interface{} {
	switch tag {
	case "toMap":
		return map[structWrapTagValue]interface{}{
			"id":      v.ID,
			"name":    v.Name,
			"surname": v.Surname,
			"":        v.NoTag,
		}
	}
	return nil
}