package struc

import (
	"go/ast"
)

type TagName string
type TagValue string
type FieldName string

type Struct struct {
	TypeName    string
	PackageName string
	Tags        map[TagName]map[FieldName]TagValue
	FieldNames  []FieldName
	TagNames    []TagName
}

func FindStructTags(file *ast.File, typeName string, tag TagName) *Struct {

	var str *Struct

	inspectRoutine := func(node ast.Node) bool {
		var typeSpec *ast.TypeSpec
		var ok bool
		typeSpec, ok = node.(*ast.TypeSpec)
		if !ok {
			return true
		}

		rawType := typeSpec.Type
		n := typeSpec.Name.Name
		if typeName != "" && n != typeName {
			return true
		}

		var structType *ast.StructType
		structType, ok = rawType.(*ast.StructType)
		if !ok {
			return true
		}

		fields := structType.Fields.List

		tags := make(map[TagName]map[FieldName]TagValue)
		fieldNames := make([]FieldName, 0, len(fields))
		tagNames := make([]TagName, 0)

		for _, field := range fields {
			for _, _fieldName := range field.Names {
				tagsValues := field.Tag.Value
				fieldTagValues, fieldTagNames := ParseTags(tagsValues)

				if tag != "" {
					_tagValue, tagValueOk := fieldTagValues[tag]
					if tagValueOk {
						fieldTagValues = map[TagName]TagValue{tag: _tagValue}
						fieldTagNames = []TagName{tag}
					} else {
						fieldTagNames = make([]TagName, 0)
					}
				}

				fldName := FieldName(_fieldName.Name)
				fieldNames = append(fieldNames, fldName)

				for _, fieldTagName := range fieldTagNames {
					fieldTagValue := fieldTagValues[fieldTagName]

					tagFields, tagFieldsOk := tags[fieldTagName]
					if !tagFieldsOk {
						tagFields = make(map[FieldName]TagValue)
						tags[fieldTagName] = tagFields
						tagNames = append(tagNames, fieldTagName)
					}

					tagFields[fldName] = fieldTagValue
				}
			}
		}

		if len(tags) > 0 {
			str = &Struct{
				TypeName:    typeName,
				PackageName: file.Name.Name,
				Tags:        tags,
				FieldNames:  fieldNames,
				TagNames:    tagNames,
			}
		}

		return false
	}
	ast.Inspect(file, inspectRoutine)

	return str

}

func ParseTags(tags string) (map[TagName]TagValue, []TagName) {
	tagNames := make([]TagName, 0)
	tagValues := make(map[TagName]TagValue)

	var prevTagPos int
	tagValueLen := len(tags)
	for pos := 0; pos < tagValueLen; pos++ {
		character := rune(tags[pos])
		switch character {
		case '`', ' ':
			prevTagPos = pos + 1
		case ':':
			_tagName := TagName(tags[prevTagPos:pos])

			//parse TagValue
			pos++

			character = rune(tags[pos])
			tagValueBorder := '"'
			findEndBorder := false
			if character == tagValueBorder {
				pos++
				findEndBorder = true
			}
			tagDelim := ' '

			var endValuePos int
			for endValuePos = pos; endValuePos < tagValueLen; endValuePos++ {
				character = rune(tags[endValuePos])
				if findEndBorder && character == tagValueBorder {
					break
				} else if character == tagDelim {
					break
				}
			}

			_tagValue := TagValue(tags[pos:endValuePos])
			tagValues[_tagName] = _tagValue
			tagNames = append(tagNames, _tagName)
			prevTagPos = endValuePos
			pos = prevTagPos

		}
	}
	return tagValues, tagNames
}
