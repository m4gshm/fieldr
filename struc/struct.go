package struc

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"reflect"
	"regexp"
	"strings"
)

type TagName string
type TagValue string
type FieldName string

type Struct struct {
	TypeName       string
	PackageName    string
	TagValueMap    map[FieldName]map[TagName]TagValue
	FieldNames     []FieldName
	TagNames       []TagName
	Constants      []string
	ConstantValues map[string]string
}

func FindStructTags(files []*ast.File, typeName string, tag TagName, tagParsers map[TagName]TagValueParser, excludeTagValues map[TagName]map[TagValue]bool, constants []string) (*Struct, error) {
	var str *Struct

	constSet := make(map[string]int, len(constants))
	for i, c := range constants {
		constSet[c] = i
	}
	constantValues := make(map[string]string, len(constants))

	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch nt := node.(type) {
			case *ast.TypeSpec:
				return handleTypeSpec(nt, typeName, tagParsers, excludeTagValues, tag, &str, file)
			case *ast.ValueSpec:
				for _, name := range nt.Names {
					n := name.Name
					_, ok := constSet[n]
					if ok {
						for _, value := range nt.Values {
							strValue, _, err := toStringValue(value)
							if err != nil {
								log.Fatalf("cons template error, const %v, error %v", n, err)
							}
							constantValues[n] = strValue

							break //only first
						}
					}
				}
				return false
			default:
				return true
			}
		})
	}

	if len(constants) != len(constantValues) {
		notFound := make([]string, 0)
		for _, constant := range constants {
			_, ok := constantValues[constant]
			if !ok {
				notFound = append(notFound, constant)
			}
		}
		return nil, errors.New("invalid const: " + strings.Join(notFound, ", "))
	}

	if str != nil {
		str.Constants = constants
		str.ConstantValues = constantValues
	}
	return str, nil

}

func toStringValue(value ast.Expr) (string, token.Token, error) {
	var strValue string
	var kind token.Token = -1
	switch vt := value.(type) {
	case *ast.BasicLit:
		strValue = vt.Value
		kind = vt.Kind
	case *ast.BinaryExpr:
		x, xKind, err := toStringValue(vt.X)
		if err != nil {
			return x, xKind, err
		}
		y, yKind, err := toStringValue(vt.Y)
		if err != nil {
			return y, yKind, err
		}
		op := vt.Op
		if xKind == yKind && xKind == token.STRING && op == token.ADD {
			xLen := len(x)
			yLen := len(y)

			if xLen == 0 {
				strValue = y
			} else if yLen == 0 {
				strValue = x
			} else {
				var xQuote string
				var yQuote string
				if xLen > 0 {
					xQuote = x[xLen-1:]
				}
				if yLen > 0 {
					yQuote = y[yLen-1:]
				}

				if xQuote == yQuote {
					strValue = x[:xLen-1] + y[1:]
				}
			}
		} else {
			strValue = x + op.String() + y
		}
		kind = yKind
	case *ast.Ident:
		strValue = vt.Name
		kind = token.IDENT
	default:
		return "", kind, fmt.Errorf("unsupported constant value part %s, type %v", value, reflect.TypeOf(value))
	}
	return strValue, kind, nil
}

func handleTypeSpec(typeSpec *ast.TypeSpec, typeName string, tagParsers map[TagName]TagValueParser, excludeTagValues map[TagName]map[TagValue]bool, tag TagName, str **Struct, file *ast.File) bool {
	rawType := typeSpec.Type
	n := typeSpec.Name.Name

	if typeName != "" && n != typeName {
		return true
	}

	structType, ok := rawType.(*ast.StructType)
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

			fieldTag := field.Tag
			var tagsValues string
			if fieldTag != nil {
				tagsValues = fieldTag.Value
			} else {
				tagsValues = ""
			}

			fieldTagValues, fieldTagNames := ParseTags(tagsValues, tagParsers, excludeTagValues)

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
		*str = &Struct{
			TypeName:    typeName,
			PackageName: file.Name.Name,
			TagValueMap: fields,
			FieldNames:  fieldNames,
			TagNames:    tagNames,
		}
	}

	return false
}

//func getTagValueTemplates() (map[TagName]*regexp.Regexp, error) {
//	jsonPattern, err := regexp.Compile(`(?P<value>[\p{L}\d]*)(,.*)*`)
//	if err != nil {
//		return nil, err
//	}
//	return map[TagName]*regexp.Regexp{
//		"json": jsonPattern,
//	}, nil
//}

func ParseTags(tags string, parsers map[TagName]TagValueParser, excludeTagValues map[TagName]map[TagValue]bool) (map[TagName]TagValue, []TagName) {
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

			parser, ok := parsers[_tagName]
			var parsedValue TagValue
			if ok {
				parsedValue = parser(tagContent)
			} else {
				parsedValue = TagValue(tagContent)
			}

			var excluded bool
			excludedValues, ok := excludeTagValues[_tagName]
			if ok {
				excluded, ok = excludedValues[parsedValue]
				excluded = excluded && ok
			}

			if !excluded {
				tagValues[_tagName] = parsedValue
				tagNames = append(tagNames, _tagName)
			}

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
