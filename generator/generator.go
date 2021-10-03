package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"
)

type Generator struct {
	Export            bool
	ExportVars        bool
	OnlyExported      bool
	ReturnRefs        bool
	WrapType          bool
	NoEmptyTag        bool
	Opts              *GenerateContentOptions
	head              bytes.Buffer
	body              bytes.Buffer
	Name              string
	used              Used
	excludedTagValues map[string]bool
	Constants         []string
}

func NewGenerator(name string, wrapType bool, refs bool, export bool, onlyExported bool, exportVars bool, noEmptyTag bool, constants []string, options *GenerateContentOptions) Generator {
	return Generator{
		Name:              name,
		WrapType:          wrapType,
		ReturnRefs:        refs,
		Export:            export,
		OnlyExported:      onlyExported,
		ExportVars:        exportVars,
		NoEmptyTag:        noEmptyTag,
		Constants:         constants,
		Opts:              options,
		excludedTagValues: make(map[string]bool),
	}
}

type GenerateContentOptions struct {
	Fields           *bool
	Tags             *bool
	FieldTagsMap     *bool
	TagValuesMap     *bool
	TagFieldsMap     *bool
	FieldTagValueMap *bool

	GetFieldValue           *bool
	GetFieldValueByTagValue *bool
	GetFieldValuesByTag     *bool
	AsMap                   *bool
	AsTagMap                *bool

	Strings *bool
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

func (g *Generator) GenerateFile(str *struc.Struct) error {
	return g.Generate(str.PackageName, str.TypeName, str.TagNames, str.FieldNames, str.TagValueMap, str.Constants, str.ConstantNames, str.ConstantValues)
}

const baseType = "string"

func (g *Generator) Generate(packageName string, typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fieldsTagValue map[struc.FieldName]map[struc.TagName]struc.TagValue, constants []string, constantNames map[string]string, constantValues map[string]string) error {

	if g.NoEmptyTag {
		for fieldName, _tagNames := range fieldsTagValue {
			for tagName, tagValue := range _tagNames {
				tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName)
				if isEmpty(tagValue) {
					g.excludedTagValues[tagValueConstName] = true
				}
			}
		}
	}

	opts := g.Opts

	if len(constants) > 0 {
		err := g.generateConstants(typeName, tagNames, fieldNames, fieldsTagValue, constants, constantNames, constantValues)
		if err != nil {
			return err
		}
	}

	genFields := *opts.Fields
	genFieldTagsMap := *opts.FieldTagsMap
	genTags := *opts.Tags
	getTagValuesMap := *opts.TagValuesMap
	genTagFieldsMap := *opts.TagFieldsMap
	getFieldTagValueMap := *opts.FieldTagValueMap

	genVars := genFields || genFieldTagsMap || genTags || getTagValuesMap || genTagFieldsMap || getFieldTagValueMap

	if genVars {
		g.writeBody("var(\n")
	}

	if genFields {
		g.generateFieldsVar(typeName, fieldNames)
	}

	if genTags {
		g.generateTagsVar(typeName, tagNames)
	}

	if genFieldTagsMap {
		g.generateFieldTagsMapVar(typeName, tagNames, fieldNames, fieldsTagValue)
	}

	if getTagValuesMap {
		g.generateTagValuesMapVar(typeName, tagNames, fieldNames, fieldsTagValue)
	}

	if genTagFieldsMap {
		g.generateTagFieldsMapVar(typeName, tagNames, fieldNames, fieldsTagValue)
	}

	if getFieldTagValueMap {
		g.generateFieldTagValueMapVar(fieldNames, tagNames, typeName, fieldsTagValue)
	}

	if genVars {
		g.writeBody(")\n")
	}

	returnRefs := g.ReturnRefs

	if *opts.GetFieldValue {
		g.generateGetFieldValueFunc(typeName, fieldNames, returnRefs)
		g.writeBody("\n")
	}
	if *opts.GetFieldValueByTagValue {
		g.generateGetFieldValueByTagValueFunc(typeName, fieldNames, tagNames, fieldsTagValue, returnRefs)
		g.writeBody("\n")
	}
	if *opts.GetFieldValuesByTag {
		g.generateGetFieldValuesByTagFunc(typeName, fieldNames, tagNames, fieldsTagValue, returnRefs)
		g.writeBody("\n")
	}
	if *opts.AsMap {
		g.generateAsMapFunc(typeName, fieldNames, returnRefs)
		g.writeBody("\n")
	}
	if *opts.AsTagMap {
		g.generateAsTagMapFunc(typeName, fieldNames, tagNames, fieldsTagValue, returnRefs)
		g.writeBody("\n")
	}

	g.generateHead(packageName, typeName, tagNames, fieldNames, fieldsTagValue, opts)

	return nil
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

	fieldConstName := g.used.fieldConstName
	tagConstName := g.used.tagConstName
	tagValueConstName := g.used.tagValueConstName

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

	if g.WrapType && *opts.Strings {
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

		ti := 0
		for _, tagName := range tagNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}
			if ti > 0 {
				varValue += ", "
			}

			tagConstName := g.getTagConstName(typeName, tagName)
			tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName)
			if g.excludedTagValues[tagValueConstName] {
				continue
			}
			varValue += tagConstName + ": " + tagValueConstName
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

		ti := 0
		for _, tagName := range tagNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}

			if ti > 0 {
				varValue += ", "
			}
			tagConstName := g.getTagConstName(typeName, tagName)
			varValue += tagConstName
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_FieldTags", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
}

func (g *Generator) generateTagValuesMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	var varValue string
	tagType := baseType
	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType

	if g.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
		tagType = g.getTagType(typeName)
		varValue = "map[" + tagType + "]" + tagValueArrayType + "{\n"
	} else {
		varValue = "map[" + tagType + "]" + tagValueArrayType + "{\n"
	}
	for _, tagName := range tagNames {
		constName := g.getTagConstName(typeName, tagName)

		if g.WrapType {
			varValue += constName + ": " + tagValueArrayType + "{"
		} else {
			varValue += constName + ": []" + baseType + "{"
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

			if ti > 0 {
				varValue += ", "
			}

			tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName)
			if g.excludedTagValues[tagValueConstName] {
				continue
			}
			varValue += tagValueConstName
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_TagValues", g.ExportVars)

	g.writeBody("%v=%v\n\n", varName, varValue)
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

		ti := 0
		for _, fieldName := range fieldNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}
			if g.isFieldExcluded(fieldName) {
				continue
			}

			if ti > 0 {
				varValue += ", "
			}
			tagConstName := g.getFieldConstName(typeName, fieldName)
			varValue += tagConstName
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

	i := 0
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		if i > 0 {
			arrayVar += ", "
		}

		constName := g.getFieldConstName(typeName, fieldName)
		arrayVar += constName
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

	for i, tagName := range tagNames {
		if i > 0 {
			arrayVar += ", "
		}
		constName := g.getTagConstName(typeName, tagName)
		arrayVar += constName
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
		for _, tagName := range tagNames {
			_, ok := fields[fieldName][tagName]
			if ok {
				tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}
				if len(caseExpr) > 0 {
					caseExpr += ", "
				}
				caseExpr += tagValueConstName
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

func (g *Generator) generateGetFieldValuesByTagFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {

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

		caseExpr := g.getTagConstName(typeName, tagName)
		fieldExpr := ""
		for _, fieldName := range fieldNames {
			if g.isFieldExcluded(fieldName) {
				continue
			}
			_, ok := fields[fieldName][tagName]
			if ok {
				if len(fieldExpr) > 0 {
					fieldExpr += ", "
				}
				fieldExpr += receiverRef + "." + string(fieldName)
			}
		}
		if len(fieldExpr) > 0 {
			funcBody += "case " + caseExpr + ":\n" +
				"return " + resultType + "{" + fieldExpr + "}\n"
		}
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.writeBody(funcBody)
}

func asRefIfNeed(receiverVar string, returnRefs bool) string {
	receiverRef := receiverVar
	if returnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
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
			_, ok := fields[fieldName][tagName]

			if ok {
				tagValueConstName := g.getTagValueConstName(typeName, tagName, fieldName)
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
	g.used.tagConstName = true
	return getTagConstName(typeName, tag, g.Export)
}

func getTagConstName(typeName string, tag struc.TagName, export bool) string {
	return goName(getTagType(typeName, export)+"_"+string(tag), export)
}

func (g *Generator) getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName) string {
	g.used.tagValueConstName = true
	export := isExport(fieldName, g.Export)
	return getTagValueConstName(typeName, tag, fieldName, export)
}

func getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, export bool) string {
	export = isExport(fieldName, export)
	return goName(getTagValueType(typeName, export)+"_"+string(tag)+"_"+string(fieldName), export)
}

func (g *Generator) getFieldConstName(typeName string, fieldName struc.FieldName) string {
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

func (g *Generator) generateConstants(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fieldsTagValue map[struc.FieldName]map[struc.TagName]struc.TagValue, constants []string, constantNames map[string]string, constantValues map[string]string) error {
	fields := make([]string, len(fieldNames))
	tags := make([]string, len(tagNames))
	fieldTags := make(map[string][]string)
	tagFields := make(map[string][]string)
	tagValues := make(map[string][]string)
	ftv := make(map[string]map[string]string)

	for i, tagName := range tagNames {
		s := string(tagName)
		tags[i] = s
		f := make([]string, 0)
		vls := make([]string, 0)
		for _, fieldName := range fieldNames {
			if g.isFieldExcluded(fieldName) {
				continue
			}
			v, ok := fieldsTagValue[fieldName][tagName]
			if ok {
				f = append(f, string(fieldName))
				vls = append(vls, string(v))
			}
		}
		tagFields[s] = f
		tagValues[s] = vls
	}

	for i, fieldName := range fieldNames {
		fld := string(fieldName)
		fields[i] = fld
		if g.isFieldExcluded(fieldName) {
			continue
		}
		t := make([]string, 0)
		for _, tagName := range tagNames {
			v, ok := fieldsTagValue[fieldName][tagName]
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
	for _, constant := range constants {
		text, ok := constantValues[constant]
		if !ok {
			continue
		}

		constName := constantNames[constant]
		if len(constName) == 0 {
			constName = goName(typeName+"_"+constant, g.Export)
		}
		constBody += constName + " = "

		add := func(first int, second int) int {
			return first + second
		}

		tmpl, err := template.New(constant).Funcs(template.FuncMap{"add": add}).Parse(text)
		if err != nil {
			return errors.Wrapf(err, "const: %s", constName)
		}

		buf := bytes.Buffer{}
		err = tmpl.Execute(&buf, data)
		if err != nil {
			return err
		}

		generatedValue := buf.String()
		constBody += generatedValue + "\n"

	}
	constBody += ")\n"
	g.writeBody(constBody)
	return nil
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
