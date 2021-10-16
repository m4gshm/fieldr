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

var (
	TagParsers    = TagValueParsers{}
	ExcludeValues = map[TagName]map[TagValue]bool{}
)

type TagName string
type TagValue string
type FieldName string

type StructModel struct {
	TypeName          string
	PackageName       string
	FieldsTagValue    map[FieldName]map[TagName]TagValue
	FieldNames        []FieldName
	TagNames          []TagName
	Constants         []string
	ConstantTemplates map[string]string
}

func FindStructTags(files []*ast.File, typeName string, includedTags map[TagName]interface{}, constants []string, constantReplacers map[string]string) (*StructModel, error) {
	var (
		str = new(StructModel)

		constantNameByTemplate = make(map[string][]string, len(constants))
		constantNames          = make([]string, len(constants))
		constantSubstitutes    = make(map[string]map[string]string, len(constants))
	)

	for i, c := range constants {
		templateVar, generatingConstant, substitutes, err := splitConstantName(c)
		if err != nil {
			return nil, err
		}
		if substitutes == nil {
			substitutes = map[string]string{}
		}
		for k, v := range constantReplacers {
			if _, ok := substitutes[k]; !ok {
				substitutes[k] = v
			}
		}

		if len(templateVar) == 0 {
			return nil, fmt.Errorf("invalid constant %s, not template var", generatingConstant)
		}

		if len(generatingConstant) == 0 {
			generatingConstant = typeName + "_" + templateVar
		}

		constantNames[i] = generatingConstant
		if len(generatingConstant) > 0 {
			constantSubstitutes[generatingConstant] = substitutes
		}
		namesByTemplates, ok := constantNameByTemplate[templateVar]
		if !ok {
			namesByTemplates = make([]string, 0)
		}
		if len(generatingConstant) > 0 {
			constantNameByTemplate[templateVar] = append(namesByTemplates, generatingConstant)
		}
	}
	constantTemplates := make(map[string]string, len(constants))

	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch nt := node.(type) {
			case *ast.TypeSpec:
				return handleTypeSpec(nt, typeName, includedTags, str, file.Name.Name)
			case *ast.ValueSpec:
				for _, name := range nt.Names {
					templateConst := name.Name
					obj := name.Obj
					isConst := obj != nil && obj.Kind == ast.Con
					if !isConst {
						continue
					}
					constNames, isTemplateConst := constantNameByTemplate[templateConst]
					if !isTemplateConst {
						continue
					}
					for _, constName := range constNames {
						for _, value := range nt.Values {
							substitutes := constantSubstitutes[constName]
							strValue, _, err := toStringValue(value, substitutes)
							if err != nil {
								log.Fatalf("cons template error, const %v, error %v", templateConst, err)
							}

							constVal := strValue
							constantTemplates[constName] = constVal
						}

						break //only first
					}

				}
				return false
			default:
				return true
			}
		})
	}

	if len(constants) != len(constantTemplates) {
		notFound := make([]string, 0)
		for _, constant := range constants {
			_, ok := constantTemplates[constant]
			if !ok {
				notFound = append(notFound, constant)
			}
		}
		return nil, errors.New("invalid const: " + strings.Join(notFound, ", "))
	}

	if str != nil {
		str.Constants = constantNames
		str.ConstantTemplates = constantTemplates
	}
	return str, nil
}

func splitConstantName(constant string) (string, string, map[string]string, error) {
	index := strings.Index(constant, ":")
	if index > 0 {
		generatingConstant := constant[:index]
		templatePart := constant[index+1:]

		index = strings.Index(templatePart, ":")
		if index > 0 {
			templateConst := templatePart[:index]
			substitutePart := templatePart[index+1:]
			replacers, err := ExtractReplacers(substitutePart)
			if err != nil {
				return "", "", nil, err
			}
			return templateConst, generatingConstant, replacers, nil
		}
		return templatePart, generatingConstant, nil, nil
	}
	return constant, "", nil, nil
}

func ExtractReplacers(substituteParts ...string) (map[string]string, error) {
	substitutes := make(map[string]string)
	for _, substitutePart := range substituteParts {
		substitutesPairs := strings.Split(substitutePart, ",")
		for _, substitutesPair := range substitutesPairs {
			key, value := extractReplacer(substitutesPair)
			if len(key) > 0 {
				if _, ok := substitutes[key]; ok {
					return nil, fmt.Errorf("duplicated replacer %v", key)
				}
				substitutes[key] = value
			}
		}
	}
	return substitutes, nil
}

func extractReplacer(replacerPair string) (string, string) {
	substitute := strings.Split(replacerPair, "=")
	replaced := ""
	replacer := ""
	if len(substitute) >= 1 {
		replaced = substitute[0]

	}
	if len(substitute) >= 2 {
		replacer = substitute[1]
	}
	return replaced, replacer
}

func toStringValue(value ast.Expr, substitutes map[string]string) (string, token.Token, error) {
	var strValue string
	var kind token.Token = -1
	switch vt := value.(type) {
	case *ast.BasicLit:
		strValue = vt.Value
		kind = vt.Kind
	case *ast.BinaryExpr:
		x, xKind, err := toStringValue(vt.X, substitutes)
		if err != nil {
			return x, xKind, err
		}
		y, yKind, err := toStringValue(vt.Y, substitutes)
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
		substitute, ok := substitutes[strValue]
		if ok {
			strValue = substitute
			if len(substitute) > 0 && substitute[0] == '"' && substitute[len(substitute)-1] == '"' {
				kind = token.STRING
			}
		}

	default:
		return "", kind, fmt.Errorf("unsupported constant value part %s, type %v", value, reflect.TypeOf(value))
	}
	return strValue, kind, nil
}

func handleTypeSpec(typeSpec *ast.TypeSpec, typeName string, includedTags map[TagName]interface{}, str *StructModel, packageName string) bool {
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

			tagValues, fieldTagNames := ParseTags(tagsValues)

			if len(includedTags) > 0 {
				filteredFieldTagValues := make(map[TagName]TagValue)
				filteredFieldTagNames := make([]TagName, 0)
				for includedTag := range includedTags {
					if _tagValue, tagValueOk := tagValues[includedTag]; tagValueOk {
						filteredFieldTagValues[includedTag] = _tagValue
						filteredFieldTagNames = append(filteredFieldTagNames, includedTag)
					}
				}
				tagValues = filteredFieldTagValues
				fieldTagNames = filteredFieldTagNames
			}

			fldName := FieldName(_fieldName.Name)

			fieldTagValues := make(map[TagName]TagValue)
			fieldNames = append(fieldNames, fldName)

			for _, fieldTagName := range fieldTagNames {
				fieldTagValue := tagValues[fieldTagName]

				tagFields, tagFieldsOk := tags[fieldTagName]
				if !tagFieldsOk {
					tagFields = make(map[FieldName]TagValue)
					tags[fieldTagName] = tagFields
					tagNames = append(tagNames, fieldTagName)
				}

				tagFields[fldName] = fieldTagValue

				fieldTagValues[fieldTagName] = fieldTagValue
			}

			if len(fieldTagValues) > 0 {
				fields[fldName] = fieldTagValues
			}
		}
	}

	*str = StructModel{
		TypeName:       typeName,
		PackageName:    packageName,
		FieldsTagValue: fields,
		FieldNames:     fieldNames,
		TagNames:       tagNames,
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

			tagContent := tags[pos:endValuePos]

			var parsedValue TagValue
			if parser, ok := TagParsers[_tagName]; ok {
				parsedValue = parser(tagContent)
			} else {
				parsedValue = TagValue(tagContent)
			}

			var excluded bool
			if excludedValues, ok := ExcludeValues[_tagName]; ok {
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
