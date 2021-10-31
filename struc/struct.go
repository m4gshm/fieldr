package struc

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
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
	TagParsers    = TagValueParsers{}
	ExcludeValues = map[TagName]map[TagValue]bool{}
)

type TagName string
type TagValue string
type FieldName string
type FieldType string

type StructModel struct {
	TypeName          string
	PackageName       string
	PackagePath       string
	FilePath          string
	FieldsTagValue    map[FieldName]map[TagName]TagValue
	FieldNames        []FieldName
	FieldsType        map[FieldName]FieldType
	TagNames          []TagName
	Constants         []string
	ConstantTemplates map[string]string
}

func FindStructTags(filePackages map[*ast.File]*packages.Package, files []*ast.File, fileSet *token.FileSet, typeName string, includedTags map[TagName]interface{}, constants []string, constantReplacers map[string]string) (*StructModel, error) {
	var (
		structModel = new(StructModel)

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
		for _, decl := range file.Decls {
			switch dt := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range dt.Specs {
					switch st := spec.(type) {
					case *ast.ValueSpec:
						extractConstants(st, constantNameByTemplate, constantSubstitutes, constantTemplates)
					case *ast.TypeSpec:
						var (
							name   = identName(st.Name)
							stType = st.Type
						)
						if structType, ok := stType.(*ast.StructType); ok && name == typeName {
							var (
								pkg      = filePackages[file]
								fileInfo = fileSet.File(file.Pos())
								filePath = fileInfo.Name()
								err      error
							)
							if structModel, err = handleStruct(structType, includedTags, typeName, filePath, pkg); err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
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

	if structModel != nil {
		structModel.Constants = constantNames
		structModel.ConstantTemplates = constantTemplates
	}
	return structModel, nil
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

func identName(ident *ast.Ident) string {
	if ident == nil {
		return ""
	}
	return ident.Name
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

type structModelBuilder struct {
	tags       map[TagName]map[FieldName]TagValue
	fields     map[FieldName]map[TagName]TagValue
	fieldNames []FieldName
	fieldsType map[FieldName]FieldType
	tagNames   []TagName

	includedTags map[TagName]interface{}
}

func (s *structModelBuilder) populateTags(fieldName FieldName, fieldTagName TagName, fieldTagValue TagValue) {
	tagFields, tagFieldsOk := s.tags[fieldTagName]
	if !tagFieldsOk {
		tagFields = make(map[FieldName]TagValue)
		s.tags[fieldTagName] = tagFields
		s.tagNames = append(s.tagNames, fieldTagName)
	}
	tagFields[fieldName] = fieldTagValue
}

func (s *structModelBuilder) populateFields(fldName FieldName, fieldTagNames []TagName, tagValues map[TagName]TagValue) {
	fieldTagValues := newFieldTagValues(fieldTagNames, tagValues)
	if len(fieldTagValues) > 0 {
		s.fields[fldName] = fieldTagValues
	}
}

func (s *structModelBuilder) populateByStruct(typeStruct *types.Struct) error {
	numFields := typeStruct.NumFields()
	for i := 0; i < numFields; i++ {
		v := typeStruct.Field(i)
		fldName := FieldName(v.Name())
		if v.IsField() {
			t := v.Type()
			if v.Embedded() {
				if err := s.populateByType(t); err != nil {
					return err
				}
			} else {
				tag := typeStruct.Tag(i)
				s.fieldNames = append(s.fieldNames, fldName)
				s.fieldsType[fldName] = FieldType(t.String())
				tagValues, fieldTagNames := parseTagValues(tag, s.includedTags)
				s.populateFields(fldName, fieldTagNames, tagValues)
				for _, fieldTagName := range fieldTagNames {
					s.populateTags(fldName, fieldTagName, tagValues[fieldTagName])
				}
			}
		}
	}
	return nil
}

func (s *structModelBuilder) populateByExpressionType(expr ast.Expr, pkg *packages.Package) error {
	typeAndVal := pkg.TypesInfo.Types[expr]
	if typeAndVal.IsType() {
		if err := s.populateByType(typeAndVal.Type); err != nil {
			return err
		}
	}
	return nil
}

func (s *structModelBuilder) populateByType(t types.Type) error {
	switch tt := t.(type) {
	case *types.Struct:
		return s.populateByStruct(tt)
	case *types.Named:
		underlying := tt.Underlying()
		return s.populateByType(underlying)
	default:
		return fmt.Errorf("unsupported type %s, type %v", tt, reflect.TypeOf(tt))
	}
}
func (s *structModelBuilder) newModel(typeName string, filePath string, structType *ast.StructType, pkg *packages.Package) (*StructModel, error) {

	if err := s.populateByExpressionType(structType, pkg); err != nil {
		return nil, err
	}

	return &StructModel{
		TypeName:       typeName,
		FilePath:       filePath,
		PackageName:    pkg.Name,
		PackagePath:    pkg.PkgPath,
		FieldsTagValue: s.fields,
		FieldNames:     s.fieldNames,
		FieldsType:     s.fieldsType,
		TagNames:       s.tagNames,
	}, nil
}

func handleStruct(structType *ast.StructType, includedTags map[TagName]interface{}, typeName string, filePath string, pkg *packages.Package) (outStructModel *StructModel, err error) {
	builder := newBuilder(includedTags)
	return builder.newModel(typeName, filePath, structType, pkg)
}

func newBuilder(includedTags map[TagName]interface{}) structModelBuilder {
	return structModelBuilder{
		tags:         map[TagName]map[FieldName]TagValue{},
		fields:       map[FieldName]map[TagName]TagValue{},
		fieldNames:   []FieldName{},
		fieldsType:   map[FieldName]FieldType{},
		tagNames:     []TagName{},
		includedTags: includedTags,
	}
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
