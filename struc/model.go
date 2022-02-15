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

	"golang.org/x/tools/go/packages"
)

const ReplaceableValueSeparator = "="
const KeyValueSeparator = ":"
const ListValuesSeparator = ","

var (
	tagParsers    = TagValueParsers{}
	excludeValues = map[TagName]map[TagValue]bool{}
)

type TagName = string
type TagValue = string
type FieldName = string
type FieldType = string

type HierarchicalModel struct {
	Model
	Nested map[FieldName]*HierarchicalModel
}

type Model struct {
	TypeName          string
	PackageName       string
	PackagePath       string
	FilePath          string
	FieldsTagValue    map[FieldName]map[TagName]TagValue
	TagsFieldValue    map[TagName]map[FieldName]TagValue
	FieldNames        []FieldName
	FieldsType        map[FieldName]FieldType
	TagNames          []TagName
	Constants         []string
	ConstantTemplates map[string]string
}

func FindStructTags(filePackages map[*ast.File]*packages.Package, files []*ast.File, fileSet *token.FileSet, typeName string, includedTags map[TagName]interface{}, constants []string, constantReplacers map[string]string) (*HierarchicalModel, error) {
	constantNameByTemplate, constantNames, constantSubstitutes, err := extractConstantNameAndTemplates(constants, constantReplacers, typeName)
	if err != nil {
		return nil, err
	}
	constantTemplates := make(map[string]string, len(constants))

	structModel := new(HierarchicalModel)
	for _, file := range files {
		var (
			filePackage = filePackages[file]
			pkg         = filePackage.Types
			fileInfo    = fileSet.File(file.Pos())
			filePath    = fileInfo.Name()
		)
		if lookup := pkg.Scope().Lookup(typeName); lookup != nil {
			if builder, err := newBuilder(pkg, nil, typeName, filePath, includedTags, handledStructs{}); err != nil {
				return nil, fmt.Errorf("new builder of %v: %w", typeName, err)
			} else if structModel, err = builder.newModel(lookup.Type()); err != nil {
				return nil, fmt.Errorf("new model of %v: %w", typeName, err)
			}
		}

		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typedSpec, ok := spec.(*ast.ValueSpec); ok {
						extractConstants(typedSpec, constantNameByTemplate, constantSubstitutes, constantTemplates)
					}
				}
			}
		}
	}

	if len(constants) != len(constantTemplates) {
		notFound := make([]string, 0)
		for _, constant := range constants {
			if _, ok := constantTemplates[constant]; !ok {
				notFound = append(notFound, constant)
			}
		}
		return nil, errors.New("invalid const: " + strings.Join(notFound, ", "))
	}

	if structModel != nil {
		structModel.Constants = constantNames
		structModel.ConstantTemplates = constantTemplates
	}
	return structModel, nil
}

func extractConstantNameAndTemplates(constants []string, constantReplacers map[string]string, typeName string) (map[string][]string, []string, map[string]map[string]string, error) {
	var (
		constantNameByTemplate = make(map[string][]string, len(constants))
		constantNames          = make([]string, len(constants))
		constantSubstitutes    = make(map[string]map[string]string, len(constants))
	)

	for i, c := range constants {
		templateVar, generatingConstant, substitutes, err := splitConstantName(c)
		if err != nil {
			return nil, nil, nil, err
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
			return nil, nil, nil, fmt.Errorf("invalid constant %s, not template var", generatingConstant)
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
	return constantNameByTemplate, constantNames, constantSubstitutes, nil
}

func extractConstants(valueSpec *ast.ValueSpec, constantNameByTemplate map[string][]string, constantSubstitutes map[string]map[string]string, constantTemplates map[string]string) {
	for _, name := range valueSpec.Names {
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
			for _, value := range valueSpec.Values {
				substitutes := constantSubstitutes[constName]
				strValue, _, err := extractConstantValue(value, substitutes)
				if err != nil {
					log.Fatalf("cons template error, const %v, error %v", templateConst, err)
				}

				constVal := strValue
				constantTemplates[constName] = constVal
			}
			break //only first
		}
	}
}

func splitConstantName(constant string) (string, string, map[string]string, error) {
	index := strings.Index(constant, KeyValueSeparator)
	if index > 0 {
		generatingConstant := constant[:index]
		templatePart := constant[index+1:]

		index = strings.Index(templatePart, KeyValueSeparator)
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
		substitutesPairs := strings.Split(substitutePart, ListValuesSeparator)
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
	substitute := strings.Split(replacerPair, ReplaceableValueSeparator)
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

func extractConstantValue(value ast.Expr, substitutes map[string]string) (string, token.Token, error) {
	var strValue string
	var kind token.Token = -1
	switch vt := value.(type) {
	case *ast.BasicLit:
		strValue = vt.Value
		kind = vt.Kind
	case *ast.BinaryExpr:
		x, xKind, err := extractConstantValue(vt.X, substitutes)
		if err != nil {
			return x, xKind, err
		}
		y, yKind, err := extractConstantValue(vt.Y, substitutes)
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

				if (xQuote == "\"" || xQuote == "`") && (yQuote == "\"" || yQuote == "`") {
					strValue = x[:xLen-1] + y[1:]
				} else {
					strValue = x + op.String() + y
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

func newFieldTagValues(fieldTagNames []TagName, tagValues map[TagName]TagValue) map[TagName]TagValue {
	fieldTagValues := make(map[TagName]TagValue, len(fieldTagNames))
	for _, fieldTagName := range fieldTagNames {
		fieldTagValues[fieldTagName] = tagValues[fieldTagName]
	}
	return fieldTagValues
}

func parseTagValues(tagsValues string, includedTags map[TagName]interface{}) (map[TagName]TagValue, []TagName) {
	tagValues, fieldTagNames := ParseTags(tagsValues)
	return filterIncludedTags(includedTags, tagValues, fieldTagNames)
}

func filterIncludedTags(includedTags map[TagName]interface{}, tagValues map[TagName]TagValue, fieldTagNames []TagName) (map[TagName]TagValue, []TagName) {
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
	return tagValues, fieldTagNames
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

			tagContent := tags[pos:endValuePos]

			var parsedValue TagValue
			if parse, ok := tagParsers[_tagName]; ok {
				parsedValue = parse(tagContent)
			} else {
				parsedValue = tagContent
			}

			var excluded bool
			if excludedValues, ok := excludeValues[_tagName]; ok {
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

func GetFieldRef(fields ...FieldName) FieldName {
	result := ""
	for _, field := range fields {
		if len(result) > 0 {
			result += "."
		}
		result += field
	}
	return result
}
