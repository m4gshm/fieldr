package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"
)

const oneLineSize = 3

type Generator struct {
	Export            bool
	ExportVars        bool
	OnlyExported      bool
	ReturnRefs        bool
	WrapType          bool
	HardcodeValues    bool
	NoEmptyTag        bool
	Compact           bool
	Opts              *GenerateContentOptions
	head              bytes.Buffer
	body              bytes.Buffer
	Name              string
	used              Used
	excludedTagValues map[string]bool
	Constants         []string
	ConstLength       int
}

func NewGenerator(name string, wrapType bool, hardcodeValues bool, refs bool, export bool, onlyExported bool,
	exportVars bool, compact bool, noEmptyTag bool, constants []string, constLength int, constReplacer string, options *GenerateContentOptions) Generator {
	return Generator{
		Name:              name,
		WrapType:          wrapType,
		HardcodeValues:    hardcodeValues,
		ReturnRefs:        refs,
		Export:            export,
		OnlyExported:      onlyExported,
		ExportVars:        exportVars,
		Compact:           compact,
		NoEmptyTag:        noEmptyTag,
		Constants:         constants,
		ConstLength:       constLength,
		Opts:              options,
		excludedTagValues: make(map[string]bool),
	}
}

type GenerateContentOptions struct {
	All bool

	Fields           *bool
	Tags             *bool
	FieldTagsMap     *bool
	TagValuesMap     *bool
	TagValues        *[]string
	TagFieldsMap     *bool
	FieldTagValueMap *bool

	GetFieldValue              *bool
	GetFieldValueByTagValue    *bool
	GetFieldValuesByTag        *[]string
	GetFieldValuesByTagGeneric *bool
	AsMap                      *bool
	AsTagMap                   *bool

	Strings  *bool
	Excludes *bool

	EnumFields    *bool
	EnumTags      *bool
	EnumTagValues *bool
}

type Used struct {
	fieldType         bool
	fieldArrayType    bool
	tagType           bool
	tagArrayType      bool
	tagValueType      bool
	tagValueArrayType bool
	tagConstName      bool
	fieldConstName    bool
	tagValueConstName bool
}

func (g *Generator) writeBody(format string, args ...interface{}) {
	fmt.Fprintf(&g.body, format, args...)
}

func (g *Generator) FormatSrc() ([]byte, error) {
	src, err := g.Src()
	if err != nil {
		return src, err
	}
	fmtSrc, err := format.Source(src)
	if err != nil {
		return src, err
	}
	return fmtSrc, nil
}

func (g *Generator) Src() ([]byte, error) {
	out := bytes.Buffer{}

	_, err := out.Write(g.head.Bytes())
	if err != nil {
		return nil, err
	}
	_, err = out.Write(g.body.Bytes())
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

const baseType = "string"

func (g *Generator) GenerateFile(str *struc.Struct) error {
	if g.NoEmptyTag {
		for fieldName, _tagNames := range str.FieldsTagValue {
			for tagName, tagValue := range _tagNames {
				tagValueConstName := g.getTagValueConstName(str.TypeName, tagName, fieldName, tagValue)
				if isEmpty(tagValue) {
					g.excludedTagValues[tagValueConstName] = true
				}
			}
		}
	}

	opts := g.Opts

	if len(str.Constants) > 0 {
		err := g.generateConstants(str)
		if err != nil {
			return err
		}
	}

	var (
		getTagValues        = *opts.TagValues
		all                 = opts.All
		genFields           = all || *opts.Fields
		genFieldTagsMap     = all || *opts.FieldTagsMap
		genTags             = all || *opts.Tags
		getTagValuesMap     = all || *opts.TagValuesMap
		genTagFieldsMap     = all || *opts.TagFieldsMap
		getFieldTagValueMap = all || *opts.FieldTagValueMap
		genVars             = all || genFields || genFieldTagsMap || genTags || getTagValuesMap || len(getTagValues) > 0 ||
			genTagFieldsMap || getFieldTagValueMap
	)

	if genVars {
		g.writeBody("var(\n")
	}

	if genFields {
		g.generateFieldsVar(str.TypeName, str.FieldNames)
	}

	if genTags {
		g.generateTagsVar(str.TypeName, str.TagNames)
	}

	if genFieldTagsMap {
		g.generateFieldTagsMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)
	}

	if all || len(getTagValues) > 0 {
		values := getTagValues
		if len(getTagValues) == 0 {
			values = getTagsValues(str.TagNames)
		}
		g.generateTagValuesVar(str.TypeName, values, str.FieldNames, str.FieldsTagValue)
	}

	if getTagValuesMap {
		g.generateTagValuesMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)
	}

	if genTagFieldsMap {
		g.generateTagFieldsMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)
	}

	if getFieldTagValueMap {
		g.generateFieldTagValueMapVar(str.FieldNames, str.TagNames, str.TypeName, str.FieldsTagValue)
	}

	if genVars {
		g.writeBody(")\n")
	}

	returnRefs := g.ReturnRefs

	if all || *opts.GetFieldValue {
		g.generateGetFieldValueFunc(str.TypeName, str.FieldNames, returnRefs)
		g.writeBody("\n")
	}
	if all || *opts.GetFieldValueByTagValue {
		g.generateGetFieldValueByTagValueFunc(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue, returnRefs)
		g.writeBody("\n")
	}

	if all || (*opts.GetFieldValuesByTagGeneric) {
		g.generateGetFieldValuesByTagFuncGeneric(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue, returnRefs)
		g.writeBody("\n")
	}

	if all || len(*opts.GetFieldValuesByTag) > 0 {

		var usedTags []struc.TagName
		if len(*opts.GetFieldValuesByTag) > 0 {
			usedTagNames := make(map[string]bool)
			for _, tagName := range *opts.GetFieldValuesByTag {
				usedTagNames[tagName] = true
			}

			usedTags = make([]struc.TagName, 0, len(usedTagNames))
			for k := range usedTagNames {
				usedTags = append(usedTags, struc.TagName(k))
			}
		} else {
			usedTags = str.TagNames
		}

		g.generateGetFieldValuesByTagFunctions(str.TypeName, str.FieldNames, usedTags, str.FieldsTagValue, returnRefs)
	}

	if all || *opts.AsMap {
		g.generateAsMapFunc(str.TypeName, str.FieldNames, returnRefs)
		g.writeBody("\n")
	}
	if all || *opts.AsTagMap {
		g.generateAsTagMapFunc(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue, returnRefs)
		g.writeBody("\n")
	}

	g.generateHead(str.PackageName, str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue, opts)

	return nil
}

func getTagsValues(names []struc.TagName) []string {
	result := make([]string, len(names))
	for i, tag := range names {
		result[i] = string(tag)
	}
	return result
}

func (g *Generator) generateHead(packageName string, typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, opts *GenerateContentOptions) {
	writer := newWriter(&g.head)

	writer("// Code generated by '%s %s'; DO NOT EDIT.\n\n", g.Name, strings.Join(os.Args[1:], " "))
	writer("package %s\n", packageName)

	fieldType := baseType
	tagType := baseType
	tagValType := baseType

	usedFieldType := g.used.fieldType
	usedTagType := g.used.tagType
	usedTagValueType := g.used.tagValueType

	if usedFieldType {
		fieldType = getFieldType(typeName, g.Export)
	}
	if usedTagType {
		tagType = getTagType(typeName, g.Export)
	}
	if usedTagValueType {
		tagValType = getTagValueType(typeName, g.Export)
	}

	if g.WrapType {
		usedTypes := usedFieldType || usedTagType || usedTagValueType

		if usedTypes {
			writer("type(\n")
		}

		if usedFieldType {
			writer("%v %v\n", fieldType, baseType)
			if g.used.fieldArrayType {
				writer("%v %v\n", arrayType(fieldType), "[]"+fieldType)
			}
		}

		if usedTagType {
			writer("%v %v\n", tagType, baseType)
			if g.used.tagArrayType {
				writer("%v %v\n", arrayType(tagType), "[]"+tagType)
			}
		}

		if usedTagValueType {
			tagValueType := tagValType
			writer("%v %v\n", tagValueType, baseType)
			if g.used.tagValueArrayType {
				writer("%v %v\n", arrayType(tagValueType), "[]"+tagValueType)
			}
		}

		if usedTypes {
			writer(")\n")
		}
	}
	fieldConstName := g.used.fieldConstName || *g.Opts.EnumFields || g.Opts.All
	tagConstName := g.used.tagConstName || *g.Opts.EnumTags || g.Opts.All
	tagValueConstName := g.used.tagValueConstName || *g.Opts.EnumTagValues || g.Opts.All

	genConst := fieldConstName || tagConstName || tagValueConstName
	if genConst {
		writer("const(\n")
	}

	if fieldConstName {
		g.generateFieldConstants(writer, typeName, fieldNames, fieldType)
		writer("\n")
	}

	if tagConstName {
		g.generateTagConstants(writer, typeName, tagNames, tagType)
		writer("\n")
	}

	if tagValueConstName {
		g.generateTagFieldConstants(writer, typeName, tagNames, fieldNames, fields, tagValType)
		writer("\n")
	}

	if genConst {
		writer(")\n")
	}

	if g.WrapType {
		if opts.All || *opts.Strings {
			if g.used.fieldArrayType {
				g.generateArrayToStringsFunc(writer, arrayType(fieldType), baseType)
				writer("\n")
			}

			if g.used.tagArrayType {
				g.generateArrayToStringsFunc(writer, arrayType(tagType), baseType)
				writer("\n")
			}

			if g.used.tagValueArrayType {
				g.generateArrayToStringsFunc(writer, arrayType(tagValType), baseType)
				writer("\n")
			}
		}

		if *opts.Excludes {
			if g.used.fieldArrayType {
				g.generateArrayToExcludesFunc(writer, true, fieldType, arrayType(fieldType))
				writer("\n")
			}

			if g.used.tagArrayType {
				g.generateArrayToExcludesFunc(writer, true, tagType, arrayType(tagType))
				writer("\n")
			}

			if g.used.tagValueArrayType {
				g.generateArrayToExcludesFunc(writer, true, tagValType, arrayType(tagValType))
				writer("\n")
			}
		}
	} else {
		if *opts.Excludes {
			g.generateArrayToExcludesFunc(writer, false, baseType, "[]"+baseType)
			writer("\n")
		}
	}
}

func newWriter(buffer *bytes.Buffer) func(format string, args ...interface{}) {
	return func(format string, args ...interface{}) {
		fmt.Fprintf(buffer, format, args...)
	}
}

func (g *Generator) getFieldType(typeName string) string {
	g.used.fieldType = true
	return getFieldType(typeName, g.Export)
}

func (g *Generator) getTagType(typeName string) string {
	g.used.tagType = true
	return getTagType(typeName, g.Export)
}

func (g *Generator) getTagValueType(typeName string) string {
	g.used.tagValueType = true
	return getTagValueType(typeName, g.Export)
}

func arrayType(baseType string) string {
	return baseType + "s"
}

func getTagValueType(typeName string, export bool) string {
	return goName(typeName+"TagValue", export)
}

func getTagType(typeName string, export bool) string {
	return goName(typeName+"Tag", export)
}

func getFieldType(typeName string, export bool) string {
	return goName(typeName+"Field", export)
}

func goName(name string, export bool) string {
	first := rune(name[0])
	if export {
		first = unicode.ToUpper(first)
	} else {
		first = unicode.ToLower(first)
	}
	result := string(first) + name[1:]
	return result
}

func camel(name string) string {
	if len(name) == 0 {
		return name
	}
	first := rune(name[0])
	first = unicode.ToUpper(first)
	result := string(first) + name[1:]
	return result
}

func (g *Generator) generateFieldTagValueMapVar(fieldNames []struc.FieldName, tagNames []struc.TagName, typeName string, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	//export := g.Export

	var varValue string
	fieldType := baseType
	tagType := baseType
	tagValueType := baseType
	if g.WrapType {
		tagType = g.getTagType(typeName)
		fieldType = g.getFieldType(typeName)
		tagValueType = g.getTagValueType(typeName)
	}
	varValue = "map[" + fieldType + "]map[" + tagType + "]" + tagValueType + "{\n"
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		fieldConstName := g.getFieldConstName(typeName, fieldName)

		varValue += fieldConstName + ": map[" + tagType + "]" + tagValueType + "{"

		compact := g.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
		if !compact {
			varValue += "\n"
		}

		ti := 0
		for _, tagName := range tagNames {
			tagVal, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}
			if compact && ti > 0 {
				varValue += ", "
			}

			tagConstName := g.getTagConstName(typeName, tagName)
			tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName, tagVal)
			if g.excludedTagValues[tagValueConstName] {
				continue
			}
			varValue += tagConstName + ": " + tagValueConstName
			if !compact {
				varValue += ",\n"
			}
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_FieldTagValue", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
}

func (g *Generator) generateFieldTagsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	fieldType := baseType
	tagArrayType := "[]" + baseType

	if g.WrapType {
		tagArrayType = g.getTagArrayType(typeName)
		fieldType = g.getFieldType(typeName)
	}

	varValue := "map[" + fieldType + "]" + tagArrayType + "{\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		fieldConstName := g.getFieldConstName(typeName, fieldName)

		if g.WrapType {
			varValue += fieldConstName + ": " + tagArrayType + "{"
		} else {
			varValue += fieldConstName + ": []" + baseType + "{"
		}

		compact := g.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
		if !compact {
			varValue += "\n"
		}

		ti := 0
		for _, tagName := range tagNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}

			if compact && ti > 0 {
				varValue += ", "
			}
			tagConstName := g.getTagConstName(typeName, tagName)
			varValue += tagConstName
			if !compact {
				varValue += ",\n"
			}
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_FieldTags", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
}

func (g *Generator) generategAmount(tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, fieldName struc.FieldName) int {
	l := 0
	for _, tagName := range tagNames {
		_, ok := fields[fieldName][tagName]
		if !ok {
			continue
		}
		l++
	}
	return l
}

func quoted(value interface{}) string {
	return "\"" + fmt.Sprintf("%v", value) + "\""
}

func (g *Generator) generateTagValuesVar(typeName string, tagNames []string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {

	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType
	if g.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
	}

	for _, tagName := range tagNames {
		varName := goName(typeName+"_TagValues_"+string(tagName), g.ExportVars)
		valueBody := g.generateTagValueBody(typeName, tagValueArrayType, fieldNames, fields, struc.TagName(tagName))
		g.writeBody("%v=%v\n\n", varName, valueBody)
	}

}

func (g *Generator) generateTagValuesMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	tagType := baseType
	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType

	if g.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
		tagType = g.getTagType(typeName)
	}

	varValue := "map[" + tagType + "]" + tagValueArrayType + "{\n"
	for _, tagName := range tagNames {
		constName := g.getTagConstName(typeName, tagName)
		valueBody := g.generateTagValueBody(typeName, tagValueArrayType, fieldNames, fields, tagName)
		varValue += constName + ": " + valueBody + ",\n"
	}
	varValue += "}"

	varName := goName(typeName+"_TagValues", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
}

func (g *Generator) generateTagValueBody(typeName string, tagValueArrayType string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagName struc.TagName) string {
	var varValue string
	if g.WrapType {
		varValue += tagValueArrayType + "{"
	} else {
		varValue += "[]" + baseType + "{"
	}

	compact := g.Compact || g.generatedAmount(fieldNames) <= oneLineSize
	if !compact {
		varValue += "\n"
	}

	ti := 0
	for _, fieldName := range fieldNames {
		tagVal, ok := fields[fieldName][tagName]
		if !ok {
			continue
		}

		if g.isFieldExcluded(fieldName) {
			continue
		}

		if compact && ti > 0 {
			varValue += ", "
		}

		tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName, tagVal)
		if g.excludedTagValues[tagValueConstName] {
			continue
		}
		varValue += tagValueConstName
		if !compact {
			varValue += ",\n"
		}
		ti++
	}

	varValue += "}"
	return varValue
}

func (g *Generator) getTagValueArrayType(tagValueType string) string {
	g.used.tagValueArrayType = true
	return arrayType(tagValueType)
}

func (g *Generator) generateTagFieldsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	tagType := baseType
	fieldArrayType := "[]" + baseType

	if g.WrapType {
		tagType = g.getTagType(typeName)
		fieldArrayType = g.getFieldArrayType(typeName)
	}

	varValue := "map[" + tagType + "]" + fieldArrayType + "{\n"

	for _, tagName := range tagNames {
		constName := g.getTagConstName(typeName, tagName)

		varValue += constName + ": " + fieldArrayType + "{"

		compact := g.Compact || g.generatedAmount(fieldNames) <= oneLineSize
		if !compact {
			varValue += "\n"
		}

		ti := 0
		for _, fieldName := range fieldNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}
			if g.isFieldExcluded(fieldName) {
				continue
			}

			if compact && ti > 0 {
				varValue += ", "
			}

			tagConstName := g.getFieldConstName(typeName, fieldName)
			varValue += tagConstName
			if !compact {
				varValue += ",\n"
			}
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_TagFields", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
}

func (g *Generator) generateTagFieldConstants(writer func(format string, args ...interface{}), typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagValueType string) {
	for i, _tagName := range tagNames {
		if i > 0 {
			writer("\n")
		}
		for _, _fieldName := range fieldNames {
			_tagValue, ok := fields[_fieldName][_tagName]
			if ok {

				isEmptyTag := isEmpty(_tagValue)

				if isEmptyTag {
					_tagValue = struc.TagValue(_fieldName)
				}

				tagValueConstName := getTagValueConstName(typeName, _tagName, _fieldName, g.Export)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}

				if g.WrapType {
					writer("%v=%v(\"%v\")", tagValueConstName, tagValueType, _tagValue)
				} else {
					writer("%v=\"%v\"", tagValueConstName, _tagValue)
				}

				if isEmptyTag {
					writer(" //empty tag")
				}
				writer("\n")
			}
		}
	}
}

func isEmpty(tagValue struc.TagValue) bool {
	return len(tagValue) == 0
}

func (g *Generator) generateFieldConstants(writer func(format string, args ...interface{}), typeName string, fieldNames []struc.FieldName, fieldType string) {
	for _, fieldName := range fieldNames {
		constName := getFieldConstName(typeName, fieldName, g.Export)
		if g.WrapType {
			writer("%v=%v(\"%v\")\n", constName, fieldType, fieldName)
		} else {
			writer("%v=\"%v\"\n", constName, fieldName)
		}
	}
}

func (g *Generator) generateTagConstants(writer func(format string, args ...interface{}), typeName string, tagNames []struc.TagName, tagType string) {
	for _, name := range tagNames {
		constName := getTagConstName(typeName, name, g.Export)
		if g.WrapType {
			writer("%v=%v(\"%v\")\n", constName, g.getTagType(typeName), name)
		} else {
			writer("%v=\"%v\"\n", constName, name)
		}
	}
}

func (g *Generator) generateFieldsVar(typeName string, fieldNames []struc.FieldName) {

	var arrayVar string
	if g.WrapType {
		arrayVar = g.getFieldArrayType(typeName) + "{"
	} else {
		arrayVar = "[]" + baseType + "{"
	}

	compact := g.Compact || g.generatedAmount(fieldNames) <= oneLineSize
	if !compact {
		arrayVar += "\n"
	}

	i := 0
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		if compact && i > 0 {
			arrayVar += ", "
		}

		constName := g.getFieldConstName(typeName, fieldName)
		arrayVar += constName
		if !compact {
			arrayVar += ",\n"
		}
		i++
	}
	arrayVar += "}"

	varNameTemplate := typeName + "_Fields"
	varName := goName(varNameTemplate, g.ExportVars)
	g.writeBody("%v=%v\n\n", varName, arrayVar)
}

func (g *Generator) getFieldArrayType(typeName string) string {
	g.used.fieldArrayType = true
	return arrayType(g.getFieldType(typeName))
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName) bool {
	return g.OnlyExported && isPrivate(fieldName)
}

func (g *Generator) generateTagsVar(typeName string, tagNames []struc.TagName) {

	tagArrayType := "[]" + baseType

	if g.WrapType {
		tagArrayType = g.getTagArrayType(typeName)
	}

	arrayVar := tagArrayType + "{"

	compact := g.Compact || len(tagNames) <= oneLineSize

	if !compact {
		arrayVar += "\n"
	}

	for i, tagName := range tagNames {
		if compact && i > 0 {
			arrayVar += ", "
		}
		constName := g.getTagConstName(typeName, tagName)
		arrayVar += constName

		if !compact {
			arrayVar += ",\n"
		}
	}
	arrayVar += "}"
	varName := goName(typeName+"_Tags", g.ExportVars)
	g.writeBody("%v=%v\n\n", varName, arrayVar)
}

func (g *Generator) getTagArrayType(typeName string) string {
	g.used.tagArrayType = true
	return arrayType(g.getTagType(typeName))
}

func (g *Generator) generateGetFieldValueFunc(typeName string, fieldNames []struc.FieldName, returnRefs bool) {

	var fieldType string
	if g.WrapType {
		fieldType = g.getFieldType(typeName)
	} else {
		fieldType = baseType
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	funcName := goName("GetFieldValue", g.Export)
	funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + fieldType + ") interface{} " +
		"{\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		fieldExpr := receiverRef + "." + string(fieldName)
		funcBody += "case " + g.getFieldConstName(typeName, fieldName) + ":\n" +
			"return " + fieldExpr + "\n"
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.writeBody(funcBody)
}

func (g *Generator) generateGetFieldValueByTagValueFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {

	var valType string
	if g.WrapType {
		valType = g.getTagValueType(typeName)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	funcName := goName("GetFieldValueByTagValue", g.Export)
	funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + valType + ") interface{} " +
		"{\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		var caseExpr string

		compact := g.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
		if !compact {
			caseExpr += "\n"
		}
		for _, tagName := range tagNames {
			tagVal, ok := fields[fieldName][tagName]
			if ok {
				tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName, tagVal)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}
				if compact && len(caseExpr) > 0 {
					caseExpr += ", "
				}
				caseExpr += tagValueConstName
				if !compact {
					caseExpr += ",\n"
				}
			}
		}
		if caseExpr != "" {
			funcBody += "case " + caseExpr + ":\n" +
				"return " + receiverRef + "." + string(fieldName) + "\n"
		}
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.writeBody(funcBody)
}

func (g *Generator) generateGetFieldValuesByTagFuncGeneric(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {

	var tagType = baseType
	if g.WrapType {
		tagType = g.getTagType(typeName)
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	resultType := "[]interface{}"

	funcName := goName("GetFieldValuesByTag", g.Export)
	funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + tagType + ") " + resultType + " " +
		"{\n" + "switch " + valVar + " {\n"
	for _, tagName := range tagNames {
		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, fieldNames, fields)

		caseExpr := g.getTagConstName(typeName, tagName)
		funcBody += "case " + caseExpr + ":\n" +
			"return " + fieldExpr + "\n"

	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.writeBody(funcBody)
}

func (g *Generator) generateGetFieldValuesByTagFunctions(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {

	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	resultType := "[]interface{}"

	for _, tagName := range tagNames {
		funcName := goName("GetFieldValuesByTag"+camel(string(tagName)), g.Export)
		funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "() " + resultType + " " +
			"{\n"

		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, fieldNames, fields)

		funcBody += "return " + fieldExpr + "\n"
		funcBody += "}\n"

		g.writeBody(funcBody)
	}
}

func (g *Generator) fieldValuesArrayByTag(receiverRef string, resultType string, tagName struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) string {
	fieldExpr := ""

	compact := g.Compact || g.generatedAmount(fieldNames) <= oneLineSize
	if !compact {
		fieldExpr += "\n"
	}

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		_, ok := fields[fieldName][tagName]
		if ok {
			if compact && len(fieldExpr) > 0 {
				fieldExpr += ", "
			}
			fieldExpr += receiverRef + "." + string(fieldName)
			if !compact {
				fieldExpr += ",\n"
			}
		}
	}
	fieldExpr = resultType + "{" + fieldExpr + "}"
	return fieldExpr
}

func (g *Generator) generatedAmount(fieldNames []struc.FieldName) int {
	l := 0
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		l++
	}
	return l
}

func asRefIfNeed(receiverVar string, returnRefs bool) string {
	receiverRef := receiverVar
	if returnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func (g *Generator) generateArrayToExcludesFunc(writer func(format string, args ...interface{}), receiver bool, typeName, arrayTypeName string) {
	funcName := goName("Excludes", g.Export)
	receiverVar := "v"
	funcDecl := "func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "(excludes ..." + typeName + ") " + arrayTypeName + " {\n"
	if !receiver {
		receiverVar = "values"
		funcDecl = "func " + funcName + " (" + receiverVar + " " + arrayTypeName + ", excludes ..." + typeName + ") " + arrayTypeName + " {\n"
	}

	writer(funcDecl +
		"	excl := make(map[" + typeName + "]interface{}, len(excludes))\n" +
		"	for _, e := range excludes {\n" +
		"		excl[e] = nil\n" +
		"	}\n" +
		"	withoutExcludes := make(" + arrayTypeName + ", 0, len(" + receiverVar + ")-len(excludes))\n" +
		"	for _, _v := range " + receiverVar + " {\n" +
		"		if _, ok := excl[_v]; !ok {\n" +
		"			withoutExcludes = append(withoutExcludes, _v)\n" +
		"		}\n" +
		"	}\n" +
		"	return withoutExcludes\n" +
		"}\n")
}

func (g *Generator) generateArrayToStringsFunc(writer func(format string, args ...interface{}), arrayTypeName string, resultType string) {
	funcName := goName("Strings", g.Export)
	receiverVar := "v"
	writer("" +
		"func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "() []" + resultType + " {\n" +
		"	strings := make([]" + resultType + ", len(v))\n" +
		"	for i, val := range " + receiverVar + " {\n" +
		"		strings[i] = string(val)\n" +
		"		}\n" +
		"		return strings\n" +
		"	}\n")
}

func (g *Generator) generateAsMapFunc(typeName string, fieldNames []struc.FieldName, returnRefs bool) {
	export := g.Export

	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	keyType := baseType
	if g.WrapType {
		keyType = g.getFieldType(typeName)
	}

	funcName := goName("AsMap", export)
	funcBody := "" +
		"func (" + receiverVar + " *" + typeName + ") " + funcName + "() map[" + keyType + "]interface{} {\n" +
		"	return map[" + keyType + "]interface{}{\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		funcBody += g.getFieldConstName(typeName, fieldName) + ": " + receiverRef + "." + string(fieldName) + ",\n"
	}
	funcBody += "" +
		"	}\n" +
		"}\n"

	g.writeBody(funcBody)
}

func (g *Generator) generateAsTagMapFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	tagValueType := baseType
	tagType := baseType
	if g.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagType = g.getTagType(typeName)
	}

	valueType := "interface{}"

	varName := "tag"

	mapType := "map[" + tagValueType + "]" + valueType

	funcName := goName("AsTagMap", g.Export)

	funcBody := "" +
		"func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + varName + " " + tagType + ") " + mapType + " {\n" +
		"switch " + varName + " {\n" +
		""

	for _, tagName := range tagNames {
		funcBody += "case " + g.getTagConstName(typeName, tagName) + ":\n" +
			"return " + mapType + "{\n"
		for _, fieldName := range fieldNames {
			if g.isFieldExcluded(fieldName) {
				continue
			}
			tagVal, ok := fields[fieldName][tagName]

			if ok {
				tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName, tagVal)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}
				funcBody += tagValueConstName + ": " + receiverRef + "." + string(fieldName) + ",\n"
			}
		}

		funcBody += "}\n"
	}
	funcBody += "" +
		"	}\n" +
		"return nil" +
		"}\n"

	g.writeBody(funcBody)
}

func (g *Generator) getTagConstName(typeName string, tag struc.TagName) string {
	if g.HardcodeValues {
		return quoted(tag)
	}
	g.used.tagConstName = true
	return getTagConstName(typeName, tag, g.Export)
}

func getTagConstName(typeName string, tag struc.TagName, export bool) string {
	return goName(getTagType(typeName, export)+"_"+string(tag), export)
}

func (g *Generator) getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, tagVal struc.TagValue) string {
	if g.HardcodeValues {
		return quoted(tagVal)
	}
	g.used.tagValueConstName = true
	export := isExport(fieldName, g.Export)
	return getTagValueConstName(typeName, tag, fieldName, export)
}

func getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, export bool) string {
	export = isExport(fieldName, export)
	return goName(getTagValueType(typeName, export)+"_"+string(tag)+"_"+string(fieldName), export)
}

func (g *Generator) getFieldConstName(typeName string, fieldName struc.FieldName) string {
	if g.HardcodeValues {
		return quoted(fieldName)
	}
	g.used.fieldConstName = true
	return getFieldConstName(typeName, fieldName, isExport(fieldName, g.Export))
}

type ConstTemplateData struct {
	Fields        []string
	Tags          []string
	FieldTags     map[string][]string
	TagValues     map[string][]string
	TagFields     map[string][]string
	FieldTagValue map[string]map[string]string
}

func (g *Generator) generateConstants(str *struc.Struct) error {
	fields := make([]string, len(str.FieldNames))
	tags := make([]string, len(str.TagNames))
	fieldTags := make(map[string][]string)
	tagFields := make(map[string][]string)
	tagValues := make(map[string][]string)
	ftv := make(map[string]map[string]string)

	for i, tagName := range str.TagNames {
		s := string(tagName)
		tags[i] = s
		f := make([]string, 0)
		vls := make([]string, 0)
		for _, fieldName := range str.FieldNames {
			if g.isFieldExcluded(fieldName) {
				continue
			}
			v, ok := str.FieldsTagValue[fieldName][tagName]
			if ok {
				f = append(f, string(fieldName))
				vls = append(vls, string(v))
			}
		}
		tagFields[s] = f
		tagValues[s] = vls
	}

	for i, fieldName := range str.FieldNames {
		fld := string(fieldName)
		fields[i] = fld
		if g.isFieldExcluded(fieldName) {
			continue
		}
		t := make([]string, 0)
		for _, tagName := range str.TagNames {
			v, ok := str.FieldsTagValue[fieldName][tagName]
			if ok {
				sv := string(v)
				if g.excludedTagValues[sv] {
					continue
				}
				tg := string(tagName)
				t = append(t, tg)
				m, ok2 := ftv[fld]
				if !ok2 {
					m = make(map[string]string)
					ftv[fld] = m
				}
				m[tg] = sv
			}

		}
		fieldTags[fld] = t
	}

	data := ConstTemplateData{
		Fields:        fields,
		Tags:          tags,
		FieldTags:     fieldTags,
		TagValues:     tagValues,
		TagFields:     tagFields,
		FieldTagValue: ftv,
	}

	constBody := "const(\n"
	for _, constName := range str.Constants {
		text, ok := str.ConstantTemplates[constName]
		if !ok {
			continue
		}
		constBody += goName(constName, g.Export) + " = "

		add := func(first int, second int) int {
			return first + second
		}

		tmpl, err := template.New(constName).Funcs(template.FuncMap{"add": add}).Parse(text)
		if err != nil {
			return errors.Wrapf(err, "const: %s", constName)
		}

		buf := bytes.Buffer{}
		err = tmpl.Execute(&buf, data)
		if err != nil {
			return err
		}

		generatedValue := buf.String()
		generatedValue, err = splitLines(generatedValue, g.ConstLength)
		if err != nil {
			return err
		}

		constBody += generatedValue + "\n"

	}
	constBody += ")\n"
	g.writeBody(constBody)
	return nil
}

func splitLines(generatedValue string, stepSize int) (string, error) {
	quotes := "\""
	if len(generatedValue) > stepSize {
		expr, err := parser.ParseExpr(generatedValue)
		if err != nil {
			return "", err
		}
		buf := bytes.Buffer{}

		tokenPos := make(map[int]token.Token)
		stringStartEnd := make(map[int]int)
		computeTokenPositions(expr, tokenPos, stringStartEnd)

		val := generatedValue

		pos := 0
		lenVal := len(val)
		for lenVal-pos > stepSize {
			prev := pos
			pos = stepSize + pos
			var (
				start        int
				end          int
				inStringPart bool
			)
			for start, end = range stringStartEnd {
				inStringPart = pos >= start && pos <= end
				if inStringPart {
					break
				}
			}

			if inStringPart {
				front := pos
				back := pos - 1
				for {
					split := -1
					if front == len(val) {
						split = len(val)
					} else if front < len(val) && val[front] == ' ' {
						split = front + 1
					} else if back >= 0 && val[back] == ' ' {
						split = back + 1
					}

					if split > -1 && split <= len(val) {
						s := val[prev:split]
						buf.WriteString(s)
						if split != len(val) {
							buf.WriteString(quotes)
							buf.WriteString(" + \n")
							buf.WriteString(quotes)
						}
						pos = split
						break
					} else {
						front++
						back--
					}
				}
			} else {
				front := pos
				back := pos - 1
				for {
					split := -1
					_, frontOk := tokenPos[front]
					_, backOk := tokenPos[back]
					if frontOk {
						split = front + 1
					} else if backOk {
						split = back + 1
					}

					if split > -1 && split <= len(val) {
						s := val[prev:split]
						buf.WriteString(s)
						if split != len(val) {
							buf.WriteString("\n")
						}
						pos = split
						break
					} else {
						front++
						back--
					}
				}
			}
		}
		if pos < lenVal {
			s := val[pos:]
			buf.WriteString(s)
		}
		generatedValue = buf.String()
	}
	return generatedValue, nil
}

func computeTokenPositions(expr ast.Expr, tokenPos map[int]token.Token, startEnd map[int]int) {
	switch et := expr.(type) {
	case *ast.BinaryExpr:
		pos := int(et.OpPos) - 1
		tokenPos[pos] = et.Op
		computeTokenPositions(et.X, tokenPos, startEnd)
		computeTokenPositions(et.Y, tokenPos, startEnd)
	case *ast.BasicLit:
		pos := int(et.ValuePos) - 1
		startEnd[pos] = pos + len(et.Value)
	}
}

func getFieldConstName(typeName string, fieldName struc.FieldName, export bool) string {
	export = isExport(fieldName, export)
	return goName(getFieldType(typeName, export)+"_"+string(fieldName), export)
}

func isPrivate(field struc.FieldName) bool {
	first, _ := utf8.DecodeRuneInString(string(field))
	return unicode.IsLower(first)
}

func isExport(fieldName struc.FieldName, export bool) bool {
	return !isPrivate(fieldName) && export
}
