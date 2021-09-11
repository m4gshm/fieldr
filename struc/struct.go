package struc

import (
	"go/ast"
	"log"
	"regexp"
	"strings"
)

type TagName string
type TagValue string
type FieldName string

type Struct struct {
	TypeName    string
	PackageName string
	Fields      map[FieldName]map[TagName]TagValue
	FieldNames  []FieldName
	TagNames    []TagName
}

type tagValueParser = func(tagValue string) TagValue

var tagParsers = map[TagName]tagValueParser{
	"json": jsonTagParser,
}

func jsonTagParser(tagContent string) TagValue {
	omitEmptySuffix := ",omitempty"
	if strings.HasSuffix(tagContent, omitEmptySuffix) {
		s := tagContent[0 : len(tagContent)-len(omitEmptySuffix)]
		return TagValue(s)
	}
	return TagValue(tagContent)

}

func regExpParser(regExpr string) tagValueParser {
	pattern, err := regexp.Compile(regExpr)
	if err != nil {
		log.Fatal(err)
	}

	return func(tagContent string) TagValue {
		return TagValue(extractTagValue(string(tagContent), pattern))
	}
}

func FindStructTags(file *ast.File, typeName string, tag TagName) (*Struct, error) {

	tagValTemplate, err := getTagValueTemplates()
	if err != nil {
		return nil, err
	}

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

		_fields := structType.Fields.List

		tags := make(map[TagName]map[FieldName]TagValue)
		fields := make(map[FieldName]map[TagName]TagValue)

		fieldNames := make([]FieldName, 0, len(_fields))
		tagNames := make([]TagName, 0)

		for _, field := range _fields {
			for _, _fieldName := range field.Names {

				tagsValues := field.Tag.Value
				fieldTagValues, fieldTagNames := ParseTags(tagsValues, tagValTemplate)

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

				fields[fldName] = make(map[TagName]TagValue)
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

					fields[fldName][fieldTagName] = fieldTagValue
				}
			}
		}

		if len(tags) > 0 {
			str = &Struct{
				TypeName:    typeName,
				PackageName: file.Name.Name,
				Fields:      fields,
				FieldNames:  fieldNames,
				TagNames:    tagNames,
			}
		}

		return false
	}
	ast.Inspect(file, inspectRoutine)

	return str, nil
}

func getTagValueTemplates() (map[TagName]*regexp.Regexp, error) {
	jsonPattern, err := regexp.Compile(`(?P<value>[\p{L}\d]*)(,.*)*`)
	if err != nil {
		return nil, err
	}
	return map[TagName]*regexp.Regexp{
		"json": jsonPattern,
	}, nil
}

func ParseTags(tags string, tagValTemplate map[TagName]*regexp.Regexp) (map[TagName]TagValue, []TagName) {
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

			tagContent := tags[pos:endValuePos]

			parser, ok := tagParsers[_tagName]
			if ok {
				tagValues[_tagName] = parser(tagContent)
			} else {
				tagValues[_tagName] = TagValue(tagContent)
			}
			tagNames = append(tagNames, _tagName)
			prevTagPos = endValuePos
			pos = prevTagPos

		}
	}
	return tagValues, tagNames
}

func extractTagValue(value string, template *regexp.Regexp) string {
	submatches := template.FindStringSubmatch(value)
	names := template.SubexpNames()
	for i, groupName := range names {
		if groupName == "value" {
			submatch := submatches[i]
			if len(submatch) == 0 {
				return submatch
			}
		}
	}
	return value
}
