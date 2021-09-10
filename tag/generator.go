package tag

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"unicode"

	"github.com/m4gshm/tag-constanter/struc"
)

type TagConstantsGenerator struct {
	Export     bool
	ExportVars bool
	ReturnRefs bool
	WrapType   bool
	buf        bytes.Buffer
	Name       string
}

func (g *TagConstantsGenerator) printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *TagConstantsGenerator) FormatSrc() ([]byte, error) {
	src := g.Src()
	fmtSrc, err := format.Source(src)
	if err != nil {
		return src, err
	}
	return fmtSrc, nil
}

func (g *TagConstantsGenerator) Src() []byte {
	return g.buf.Bytes()
}

func (g *TagConstantsGenerator) GenerateFile(str *struc.Struct) {
	g.Generate(str.PackageName, str.TypeName, str.TagNames, str.FieldNames, str.Fields)
}

const baseType = "string"

func (g *TagConstantsGenerator) Generate(packageName string, typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName,
	fields map[struc.FieldName]map[struc.TagName]struc.TagValue,
) {

	g.printf("// Code generated by '%s %s'; DO NOT EDIT.\n\n", g.Name, strings.Join(os.Args[1:], " "))
	g.printf("package %s\n", packageName)

	if g.WrapType {
		g.printf("type(\n")

		fieldType := getFieldType(typeName, g.Export)
		tagType := getTagType(typeName, g.Export)
		tagValueType := getTagValueType(typeName, g.Export)

		g.printf("%v %v\n", fieldType, baseType)
		g.printf("%v %v\n", arrayType(fieldType), "[]"+fieldType)
		g.printf("%v %v\n", tagType, baseType)
		g.printf("%v %v\n", arrayType(tagType), "[]"+tagType)
		g.printf("%v %v\n", tagValueType, baseType)
		g.printf("%v %v\n", arrayType(tagValueType), "[]"+tagValueType)

		g.printf(")\n")
	}

	g.printf("const(\n")

	g.generateFieldConstants(typeName, fieldNames)
	g.printf("\n")
	g.generateTagConstants(typeName, tagNames)
	g.printf("\n")
	g.generateTagFieldConstants(typeName, tagNames, fieldNames, fields)

	g.printf(")\n")

	g.printf("var(\n")

	g.generateFieldsArrayVar(typeName, fieldNames)
	g.generateTagsArrayVar(typeName, tagNames)

	g.generateTagsByFieldsMapVar(typeName, fieldNames, fields)

	g.generateTagValuesByTagMapVar(typeName, tagNames, fieldNames, fields)

	g.generateTagFieldsByTagMapVar(typeName, tagNames, fieldNames, fields)

	g.generateTagsValuesByFieldMapVar(fieldNames, typeName, fields)

	g.printf(")\n")

	if g.WrapType {
		g.generateArrayToStringsFunc(arrayType(getFieldType(typeName, g.Export)))
		g.printf("\n")
		g.generateArrayToStringsFunc(arrayType(getTagType(typeName, g.Export)))
		g.printf("\n")
		g.generateArrayToStringsFunc(arrayType(getTagValueType(typeName, g.Export)))
		g.printf("\n")
	}

	returnRefs := g.ReturnRefs

	g.generateGetValueByFieldFunc(typeName, fieldNames, returnRefs)
	g.printf("\n")

	g.generateGetFieldValueByTagFunc(typeName, fieldNames, tagNames, fields, returnRefs)
	g.printf("\n")
	g.generateAsMapFunc(typeName, fieldNames, returnRefs)

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

func (g *TagConstantsGenerator) generateTagsValuesByFieldMapVar(
	fieldNames []struc.FieldName, typeName string, fields map[struc.FieldName]map[struc.TagName]struc.TagValue,
) {
	var varValue string
	if g.WrapType {
		varValue = "map[" + getFieldType(typeName, g.Export) + "]map[" + getTagType(typeName, g.Export) + "]" + getTagValueType(typeName, g.Export) + "{\n"
	} else {
		varValue = "map[string]map[string]string{\n"
	}
	for _, fieldName := range fieldNames {
		fieldConstName := getFieldConstName(typeName, fieldName, g.Export)
		if g.WrapType {
			varValue += fieldConstName + ": map[" + getTagType(typeName, g.Export) + "]" + getTagValueType(typeName, g.Export) + "{"
		} else {
			varValue += fieldConstName + ": map[string]string{"
		}

		fieldTags := fields[fieldName]

		ti := 0
		for fieldTag := range fieldTags {
			if ti > 0 {
				varValue += ", "
			}

			tagConstName := getTagConstName(typeName, fieldTag, g.Export)
			varValue += tagConstName + ": " + getTagValueConstName(typeName, fieldTag, fieldName, g.Export)
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_Field_Tag_Value", g.ExportVars)

	g.printf("%v=%v\n\n", varName, varValue)
}

func (g *TagConstantsGenerator) generateTagsByFieldsMapVar(typeName string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	var varValue string
	if g.WrapType {
		varValue = "map[" + getFieldType(typeName, g.Export) + "]" + arrayType(getTagType(typeName, g.Export)) + "{\n"
	} else {
		varValue = "map[string][]string{\n"
	}
	for _, fieldName := range fieldNames {
		fieldConstName := getFieldConstName(typeName, fieldName, g.Export)

		if g.WrapType {
			varValue += fieldConstName + ": " + arrayType(getTagType(typeName, g.Export)) + "{"
		} else {
			varValue += fieldConstName + ": []string{"
		}

		fieldTags := fields[fieldName]

		ti := 0
		for fieldTag := range fieldTags {
			if ti > 0 {
				varValue += ", "
			}
			tagConstName := getTagConstName(typeName, fieldTag, g.Export)
			varValue += tagConstName
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_Field_Tags", g.ExportVars)

	g.printf("%v=%v\n\n", varName, varValue)
}

func (g *TagConstantsGenerator) generateTagValuesByTagMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	var varValue string
	if g.WrapType {
		varValue = "map[" + getTagType(typeName, g.Export) + "]" + arrayType(getTagValueType(typeName, g.Export)) + "{\n"
	} else {
		varValue = "map[string][]string{\n"
	}
	for _, tagName := range tagNames {
		constName := getTagConstName(typeName, tagName, g.Export)

		if g.WrapType {
			varValue += constName + ": " + arrayType(getTagValueType(typeName, g.Export)) + "{"
		} else {
			varValue += constName + ": []string{"
		}

		//tagValues := tags[tagName]

		ti := 0
		for _, field := range fieldNames {

			_, ok := fields[field][tagName]

			//_, ok := tagValues[field]
			if !ok {
				continue
			}

			if ti > 0 {
				varValue += ", "
			}
			tagConstName := getTagValueConstName(typeName, tagName, field, g.Export)
			varValue += tagConstName
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_Tag_Values", g.ExportVars)

	g.printf("%v=%v\n\n", varName, varValue)
}

func (g *TagConstantsGenerator) generateTagFieldsByTagMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) {
	var varValue string
	if g.WrapType {
		varValue = "map[" + getTagType(typeName, g.Export) + "]" + arrayType(getFieldType(typeName, g.Export)) + "{\n"
	} else {
		varValue = "map[string][]string{\n"
	}
	for _, tagName := range tagNames {
		constName := getTagConstName(typeName, tagName, g.Export)

		if g.WrapType {
			varValue += constName + ": " + arrayType(getFieldType(typeName, g.Export)) + "{"
		} else {
			varValue += constName + ": []string{"
		}

		ti := 0
		for _, field := range fieldNames {
			_, ok := fields[field][tagName]
			if !ok {
				continue
			}

			if ti > 0 {
				varValue += ", "
			}
			tagConstName := getFieldConstName(typeName, field, g.Export)
			varValue += tagConstName
			ti++
		}

		varValue += "},\n"
	}
	varValue += "}"

	varName := goName(typeName+"_Tag_Fields", g.ExportVars)

	g.printf("%v=%v\n\n", varName, varValue)
}

func (g *TagConstantsGenerator) generateTagFieldConstants(
	typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName,
	fields map[struc.FieldName]map[struc.TagName]struc.TagValue,
) {
	for i, _tagName := range tagNames {
		if i > 0 {
			g.printf("\n")
		}
		for _, _fieldName := range fieldNames {
			_tagValue, ok := fields[_fieldName][_tagName]
			if ok {
				constName := getTagValueConstName(typeName, _tagName, _fieldName, g.Export)
				if g.WrapType {
					g.printf("%v=%v(\"%v\")\n", constName, getTagValueType(typeName, g.Export), _tagValue)
				} else {
					g.printf("%v=\"%v\"\n", constName, _tagValue)
				}
			}
		}
	}
}

func (g *TagConstantsGenerator) generateFieldConstants(typeName string, fieldNames []struc.FieldName) {
	for _, name := range fieldNames {
		constName := getFieldConstName(typeName, name, g.Export)
		if g.WrapType {
			g.printf("%v=%v(\"%v\")\n", constName, getFieldType(typeName, g.Export), name)
		} else {
			g.printf("%v=\"%v\"\n", constName, name)
		}
	}
}

func (g *TagConstantsGenerator) generateTagConstants(typeName string, tagNames []struc.TagName) {
	for _, name := range tagNames {
		constName := getTagConstName(typeName, name, g.Export)
		if g.WrapType {
			g.printf("%v=%v(\"%v\")\n", constName, getTagType(typeName, g.Export), name)
		} else {
			g.printf("%v=\"%v\"\n", constName, name)
		}
	}
}

func (g *TagConstantsGenerator) generateFieldsArrayVar(typeName string, fieldNames []struc.FieldName) {
	var arrayVar string
	if g.WrapType {
		arrayVar = arrayType(getFieldType(typeName, g.Export)) + "{"
	} else {
		arrayVar = "[]string{"
	}

	for i, fieldName := range fieldNames {
		if i > 0 {
			arrayVar += ", "
		}
		constName := getFieldConstName(typeName, fieldName, g.Export)
		arrayVar += constName
	}
	arrayVar += "}"
	varName := goName(typeName+"_Fields", g.ExportVars)
	g.printf("%v=%v\n\n", varName, arrayVar)
}

func (g *TagConstantsGenerator) generateTagsArrayVar(typeName string, tagNames []struc.TagName) {
	var arrayVar string
	if g.WrapType {
		arrayVar = arrayType(getTagType(typeName, g.Export)) + "{"
	} else {
		arrayVar = "[]string{"
	}

	for i, tagName := range tagNames {
		if i > 0 {
			arrayVar += ", "
		}
		constName := getTagConstName(typeName, tagName, g.Export)
		arrayVar += constName
	}
	arrayVar += "}"
	varName := goName(typeName+"_Tags", g.ExportVars)
	g.printf("%v=%v\n\n", varName, arrayVar)
}

func (g *TagConstantsGenerator) generateGetValueByFieldFunc(typeName string, fieldNames []struc.FieldName, returnRefs bool) {

	var valType string
	if g.WrapType {
		valType = getFieldType(typeName, g.Export)
	} else {
		valType = "string"
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	funcName := goName("FieldValue", g.Export)
	funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + valType + ") interface{} " +
		"{\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		fieldExpr := receiverRef + "." + string(fieldName)
		funcBody += "case " + getFieldConstName(typeName, fieldName, g.Export) + ":\n" +
			"return " + fieldExpr + "\n"
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.printf(funcBody)
}

func (g *TagConstantsGenerator) generateGetFieldValueByTagFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, returnRefs bool) {

	var valType string
	if g.WrapType {
		valType = getTagValueType(typeName, g.Export)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	funcName := goName("FieldValueByTagValue", g.Export)
	funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + valType + ") interface{} " +
		"{\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		fieldExpr := receiverRef + "." + string(fieldName)

		var caseExpr string
		for _, tagName := range tagNames {
			_, ok := fields[fieldName][tagName]
			if ok {
				if len(caseExpr) > 0 {
					caseExpr += ", "
				}
				caseExpr += getTagValueConstName(typeName, tagName, fieldName, g.Export)
			}
		}
		funcBody += "case " + caseExpr + ":\n" +
			"return " + fieldExpr + "\n"
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	g.printf(funcBody)
}

func asRefIfNeed(receiverVar string, returnRefs bool) string {
	receiverRef := receiverVar
	if returnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func (g *TagConstantsGenerator) generateArrayToStringsFunc(arrayTypeName string) {
	funcName := goName("Strings", g.Export)
	g.printf("" +
		"func (v " + arrayTypeName + ") " + funcName + "() []string {\n" +
		"	strings := make([]string, 0, len(v))\n" +
		"	for i, v := range v {\n" +
		"		strings[i] = string(v)\n" +
		"		}\n" +
		"		return strings\n" +
		"	}\n")
}

func (g *TagConstantsGenerator) generateAsMapFunc(typeName string, fieldNames []struc.FieldName, returnRefs bool) {
	receiverVar := "v"
	receiverRef := asRefIfNeed(receiverVar, returnRefs)

	fieldType := baseType
	if g.WrapType {

		fieldType = getFieldType(typeName, g.Export)
	}

	funcName := goName("AsMap", g.Export)
	funcBody := "" +
		"func (" + receiverVar + " *" + typeName + ") " + funcName + "() map[" + fieldType + "]interface{} {\n" +
		"	return map[" + fieldType + "]interface{}{\n"

	for _, fieldName := range fieldNames {
		funcBody += getFieldConstName(typeName, fieldName, g.Export) + ": " + receiverRef + "." + string(fieldName) + ",\n"
	}
	funcBody += "" +
		"	}\n" +
		"}"

	g.printf(funcBody)
}

func getTagConstName(typeName string, tag struc.TagName, export bool) string {
	return goName(typeName+"_"+string(tag), export)
}

func getTagValueConstName(typeName string, tag struc.TagName, field struc.FieldName, export bool) string {
	return goName(typeName+"_"+string(tag)+"_"+string(field), export)
}

func getFieldConstName(typeName string, fieldName struc.FieldName, export bool) string {
	return goName(typeName+"_"+string(fieldName), export)
}
