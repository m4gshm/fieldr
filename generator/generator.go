package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"sort"
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
	body              *bytes.Buffer
	Name              string
	used              Used
	excludedTagValues map[string]bool
	Constants         []string
	ConstLength       int
	OutBuildTags      string

	constNames         []string
	constValues        map[string]string
	constComments      map[string]string
	varNames           []string
	varValues          map[string]string
	typeNames          []string
	typeValues         map[string]string
	funcNames          []string
	funcValues         map[string]string
	receiverNames      []string
	receiverFuncs      map[string][]string
	receiverFuncValues map[string]map[string]string
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
	fmt.Fprintf(g.body, format, args...)
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

	_, err := out.Write(g.body.Bytes())
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

const baseType = "string"

func (g *Generator) GenerateFile(str *struc.Struct, file *ast.File, info *token.File) error {
	g.excludedTagValues = make(map[string]bool)
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

	g.constNames = make([]string, 0)
	g.constValues = make(map[string]string)
	g.constComments = make(map[string]string)
	g.varNames = make([]string, 0)
	g.varValues = make(map[string]string)
	g.typeNames = make([]string, 0)
	g.typeValues = make(map[string]string)
	g.funcNames = make([]string, 0)
	g.funcValues = make(map[string]string)
	g.receiverNames = make([]string, 0)
	g.receiverFuncs = make(map[string][]string)
	g.receiverFuncValues = make(map[string]map[string]string)

	if err := g.generateConstants(str); err != nil {
		return err
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
	)

	if genFields {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldsVar(str.TypeName, str.FieldNames)); err != nil {
			return err
		}
	}

	if genTags {
		g.addVarDelim()
		if err := g.addVar(g.generateTagsVar(str.TypeName, str.TagNames)); err != nil {
			return err
		}
	}

	if genFieldTagsMap {
		g.addVarDelim()
		if err := g.addVar(
			g.generateFieldTagsMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || len(getTagValues) > 0 {
		g.addVarDelim()
		values := getTagValues
		if len(getTagValues) == 0 {
			values = getTagsValues(str.TagNames)
		}
		vars, bodies, err := g.generateTagValuesVar(str.TypeName, values, str.FieldNames, str.FieldsTagValue)
		if err != nil {
			return err
		}
		for _, varName := range vars {
			if err = g.addVar(varName, bodies[varName]); err != nil {
				return err
			}
		}
	}

	if getTagValuesMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagValuesMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if genTagFieldsMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagFieldsMapVar(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if getFieldTagValueMap {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldTagValueMapVar(str.FieldNames, str.TagNames, str.TypeName, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || *opts.GetFieldValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueFunc(str.TypeName, str.FieldNames)); err != nil {
			return err
		}
	}
	if all || *opts.GetFieldValueByTagValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueByTagValueFunc(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || (*opts.GetFieldValuesByTagGeneric) {
		if err := g.addReceiverFunc(g.generateGetFieldValuesByTagFuncGeneric(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue)); err != nil {
			return err
		}
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

		funcNames, funcBodies, err := g.generateGetFieldValuesByTagFunctions(str.TypeName, str.FieldNames, usedTags, str.FieldsTagValue)
		if err != nil {
			return err
		}

		for _, funcName := range funcNames {
			funcBody := funcBodies[funcName]
			if err = g.addReceiverFunc(str.TypeName, funcName, funcBody); err != nil {
				return err
			}
		}
	}

	if all || *opts.AsMap {
		if err := g.addReceiverFunc(g.generateAsMapFunc(str.TypeName, str.FieldNames)); err != nil {
			return err
		}
	}
	if all || *opts.AsTagMap {
		if err := g.addReceiverFunc(g.generateAsTagMapFunc(str.TypeName, str.FieldNames, str.TagNames, str.FieldsTagValue)); err != nil {
			return err
		}
	}

	if err := g.generateHead(str.TypeName, str.TagNames, str.FieldNames, str.FieldsTagValue, opts); err != nil {
		return err
	}

	rewrite := true
	if file != nil {
		rewrite = false
		for _, comment := range file.Comments {
			pos := comment.Pos()
			base := info.Base()
			firstComment := int(pos) == base
			if firstComment {
				text := comment.Text()
				generatedMarker := g.generatedMarker()
				generated := strings.HasPrefix(text, generatedMarker)
				rewrite = generated
				break
			}
		}
	}

	if rewrite {
		g.body = &bytes.Buffer{}
		g.writeHead(str)

		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeReceiverFunctions()
		g.writeFunctions()
	} else {

		//injects

		base := info.Base()
		chunkVals := make(map[int]map[int]string)

		for _, decl := range file.Decls {
			switch dt := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range dt.Specs {
					switch st := spec.(type) {
					case *ast.ValueSpec:
						names := st.Names
						values := st.Values
						if len(names) > 0 && len(values) > 0 {
							objectName := names[0]
							value := values[0]
							start := int(value.Pos()) - base
							end := int(value.End()) - base

							var generatingValues map[string]string
							switch dt.Tok {
							case token.TYPE:
								generatingValues = g.typeValues
							case token.CONST:
								generatingValues = g.constValues
							case token.VAR:
								generatingValues = g.varValues
							}
							name := objectName.Name
							if newValue, found := generatingValues[name]; found {
								chunkVals[start] = map[int]string{end: newValue}
								delete(generatingValues, name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				start := int(dt.Pos()) - base
				end := int(dt.End()) - base
				name := dt.Name.Name
				recv := dt.Recv
				if recv != nil {
					list := recv.List
					if len(list) > 0 {
						field := list[0]
						typ := field.Type
						receiverName := getReceiverName(typ)
						recFuncs, hasFuncs := g.receiverFuncValues[receiverName]
						if hasFuncs {
							funcDecl, hasFuncDecl := recFuncs[name]
							if hasFuncDecl {
								chunkVals[start] = map[int]string{end: funcDecl}
								delete(recFuncs, name)
							}
						}
					}
				} else {
					funcDecl, hasFuncDecl := g.funcValues[name]
					if hasFuncDecl {
						chunkVals[start] = map[int]string{end: funcDecl}
						delete(g.funcValues, name)
					}
				}
			}
		}

		chunkPos := getSortedChunks(chunkVals)

		name := info.Name()
		fileBytes, err := ioutil.ReadFile(name)
		if err != nil {
			return err
		}
		fileContent := string(fileBytes)

		newFileContent := rewriteChunks(chunkPos, chunkVals, fileContent)

		g.body = bytes.NewBufferString(newFileContent)

		//write not injected
		g.filterInjected()

		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeReceiverFunctions()
		g.writeFunctions()

	}

	return nil
}

func getReceiverName(typ ast.Expr) string {
	switch tt := typ.(type) {
	case *ast.StarExpr:
		return getReceiverName(tt.X)
	case *ast.Ident:
		return tt.Name
	}
	return ""
}

func rewriteChunks(sortedPos []int, chunkVals map[int]map[int]string, fileContent string) string {
	newFileContent := ""
	start := 0
	for _, end := range sortedPos {
		for j, value := range chunkVals[end] {
			prefix := fileContent[start:end]
			newFileContent += prefix + value
			start = j
		}
	}
	prefix := fileContent[start:]
	newFileContent += prefix
	return newFileContent
}

func getSortedChunks(chunkVals map[int]map[int]string) []int {
	chunkPos := make([]int, 0)
	for start := range chunkVals {
		chunkPos = append(chunkPos, start)
	}
	sortedChunksPos := sort.Ints
	sortedChunksPos(chunkPos)
	return chunkPos
}

func (g *Generator) writeHead(str *struc.Struct) {
	g.writeBody("// %s %s'; DO NOT EDIT.\n\n", g.generatedMarker(), strings.Join(os.Args[1:], " "))
	g.writeBody(g.OutBuildTags)
	g.writeBody("package %s\n", str.PackageName)
}

func (g *Generator) writeConstants() {
	names := g.constNames
	values := g.constValues
	comments := g.constComments
	if len(names) > 0 {
		g.writeBody("const(\n")
	}
	for _, name := range names {
		if len(name) == 0 {
			g.writeBody("\n")
			continue
		}
		value := values[name]
		g.writeBody("%v=%v", name, value)
		if comment, ok := comments[name]; ok {
			g.writeBody(comment)
		}
		g.writeBody("\n")
	}
	if len(names) > 0 {
		g.writeBody(")\n")
	}
}

func (g *Generator) writeVars() {
	names := g.varNames
	values := g.varValues
	if len(names) > 0 {
		g.writeBody("var(\n")
	}
	for _, name := range names {
		if len(name) == 0 {
			g.writeBody("\n")
			continue
		}
		value := values[name]
		g.writeBody("%v=%v", name, value)
		g.writeBody("\n")
	}
	if len(names) > 0 {
		g.writeBody(")\n")
	}
}

func (g *Generator) writeFunctions() {
	names := g.funcNames
	values := g.funcValues

	for _, name := range names {
		value := values[name]
		g.writeBody(value)
		g.writeBody("\n")
	}
}

func (g *Generator) writeReceiverFunctions() {
	receiverNames := g.receiverNames
	values := g.receiverFuncValues

	for _, receiverName := range receiverNames {
		funcNames := g.receiverFuncs[receiverName]
		for _, funcName := range funcNames {
			value := values[receiverName][funcName]
			g.writeBody(value)
			g.writeBody("\n")
		}
	}
}

func getTagsValues(names []struc.TagName) []string {
	result := make([]string, len(names))
	for i, tag := range names {
		result[i] = string(tag)
	}
	return result
}

func (g *Generator) generateHead(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, opts *GenerateContentOptions) error {

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
		if usedFieldType {
			g.addType(fieldType, baseType)
			if g.used.fieldArrayType {
				g.addType(arrayType(fieldType), "[]"+fieldType)
			}
		}

		if usedTagType {
			g.addType(tagType, baseType)
			if g.used.tagArrayType {
				g.addType(arrayType(tagType), "[]"+tagType)
			}
		}

		if usedTagValueType {
			tagValueType := tagValType
			g.addType(tagValueType, baseType)
			if g.used.tagValueArrayType {
				g.addType(arrayType(tagValueType), "[]"+tagValueType)
			}
		}
	}

	fieldConstName := g.used.fieldConstName || *g.Opts.EnumFields || g.Opts.All
	tagConstName := g.used.tagConstName || *g.Opts.EnumTags || g.Opts.All
	tagValueConstName := g.used.tagValueConstName || *g.Opts.EnumTagValues || g.Opts.All

	if fieldConstName {
		if err := g.generateFieldConstants(typeName, fieldType, fieldNames); err != nil {
			return err
		}
	}

	if tagConstName {
		if err := g.generateTagConstants(typeName, tagType, tagNames); err != nil {
			return err
		}
	}

	if tagValueConstName {
		if err := g.generateTagFieldConstants(typeName, tagNames, fieldNames, fields, tagValType); err != nil {
			return err
		}
	}

	if g.WrapType {
		if opts.All || *opts.Strings {
			if g.used.fieldArrayType {
				if err := g.addReceiverFunc(g.generateArrayToStringsFunc(arrayType(fieldType), baseType)); err != nil {
					return err
				}
			}

			if g.used.tagArrayType {
				if err := g.addReceiverFunc(g.generateArrayToStringsFunc(arrayType(tagType), baseType)); err != nil {
					return err
				}
			}

			if g.used.tagValueArrayType {
				if err := g.addReceiverFunc(g.generateArrayToStringsFunc(arrayType(tagValType), baseType)); err != nil {
					return err
				}
			}
		}

		if *opts.Excludes {
			if g.used.fieldArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, fieldType, arrayType(fieldType))
				if err := g.addReceiverFunc(fieldType, funcName, funcBody); err != nil {
					return err
				}
			}

			if g.used.tagArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagType, arrayType(tagType))
				if err := g.addReceiverFunc(tagType, funcName, funcBody); err != nil {
					return err
				}
			}

			if g.used.tagValueArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagValType, arrayType(tagValType))
				if err := g.addReceiverFunc(tagValType, funcName, funcBody); err != nil {
					return err
				}
			}
		}
	} else {
		if *opts.Excludes {
			funcName, funcBody := g.generateArrayToExcludesFunc(false, baseType, "[]"+baseType)
			if err := g.addFunc(funcName, funcBody); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Generator) writeTypes() {
	names := g.typeNames
	values := g.typeValues

	if len(names) > 0 {
		g.writeBody("type(\n")
	}

	for _, name := range names {
		value := values[name]
		g.writeBody("%v %v\n", name, value)
	}

	if len(names) > 0 {
		g.writeBody(")\n")
	}
}

func (g *Generator) addType(typeName string, typeValue string) {
	g.typeNames = append(g.typeNames, typeName)
	g.typeValues[typeName] = typeValue
}

func (g *Generator) generatedMarker() string {
	return fmt.Sprintf("Code generated by '%s", g.Name)
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

func (g *Generator) generateFieldTagValueMapVar(fieldNames []struc.FieldName, tagNames []struc.TagName, typeName string, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string) {
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
	return varName, varValue
}

func (g *Generator) generateFieldTagsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string) {
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
	return varName, varValue
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

func (g *Generator) generateTagValuesVar(typeName string, tagNames []string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) ([]string, map[string]string, error) {

	vars := make([]string, 0)
	varValues := make(map[string]string)
	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType
	if g.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
	}

	for _, tagName := range tagNames {
		varName := goName(typeName+"_TagValues_"+string(tagName), g.ExportVars)
		valueBody := g.generateTagValueBody(typeName, tagValueArrayType, fieldNames, fields, struc.TagName(tagName))
		vars = append(vars, varName)
		if _, ok := varValues[varName]; !ok {
			varValues[varName] = valueBody
		} else {
			return nil, nil, errors.Errorf("duplicated var %s", varName)
		}
	}

	return vars, varValues, nil
}

func (g *Generator) generateTagValuesMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string) {
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

	return varName, varValue
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

func (g *Generator) generateTagFieldsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string) {
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

	return varName, varValue
}

func (g *Generator) generateTagFieldConstants(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagValueType string) error {
	g.addConstDelim()
	for _, _tagName := range tagNames {
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

				constVal := g.getConstValue(tagValueType, string(_tagValue))
				if err := g.addConst(tagValueConstName, constVal); err != nil {
					return err
				}

				if isEmptyTag {
					g.constComments[tagValueConstName] = " //empty tag"
				}
			}
		}
	}
	return nil
}

func isEmpty(tagValue struc.TagValue) bool {
	return len(tagValue) == 0
}

func (g *Generator) generateFieldConstants(typeName string, fieldType string, fieldNames []struc.FieldName) error {
	g.addConstDelim()
	for _, fieldName := range fieldNames {
		constName := getFieldConstName(typeName, fieldName, g.Export)
		constVal := g.getConstValue(fieldType, string(fieldName))
		if err := g.addConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateTagConstants(typeName string, tagType string, tagNames []struc.TagName) error {
	g.addConstDelim()
	for _, name := range tagNames {
		constName := getTagConstName(typeName, name, g.Export)
		constVal := g.getConstValue(tagType, string(name))
		if err := g.addConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil

}

func (g *Generator) addConstDelim() {
	if len(g.constNames) > 0 {
		g.constNames = append(g.constNames, "")
	}
}

func (g *Generator) addConst(constName, constValue string) error {
	if _, ok := g.constValues[constName]; ok {
		return errors.Errorf("duplicated constant %v", constName)
	}
	g.constNames = append(g.constNames, constName)
	g.constValues[constName] = constValue
	return nil
}

func (g *Generator) getConstValue(typ string, value string) (constValue string) {
	if g.WrapType {
		return fmt.Sprintf("%v(\"%v\")", typ, value)
	}
	return fmt.Sprintf("\"%v\"", value)
}

func (g *Generator) addVarDelim() {
	if len(g.varNames) > 0 {
		g.varNames = append(g.varNames, "")
	}
}

func (g *Generator) addVar(varName, varValue string) error {
	if _, ok := g.varValues[varName]; ok {
		return errors.Errorf("duplicated var %v", varName)
	}
	g.varNames = append(g.varNames, varName)
	g.varValues[varName] = varValue
	return nil
}

func (g *Generator) addFunc(funcName, funcValue string) error {
	if _, ok := g.funcValues[funcName]; ok {
		return errors.Errorf("duplicated func %v", funcName)
	}
	g.funcNames = append(g.funcNames, funcName)
	g.funcValues[funcName] = funcValue
	return nil
}

func (g *Generator) addReceiverFunc(receiverName, funcName, funcValue string) error {
	funcs, ok := g.receiverFuncs[receiverName]
	if !ok {
		g.receiverNames = append(g.receiverNames, receiverName)

		funcs = make([]string, 0)
		g.receiverFuncs[receiverName] = funcs
		g.receiverFuncValues[receiverName] = make(map[string]string)
	}

	if _, ok = g.receiverFuncValues[receiverName][funcName]; ok {
		return errors.Errorf("duplicated receiver's func %v.%v", receiverName, funcName)
	}

	g.receiverFuncs[receiverName] = append(funcs, funcName)
	g.receiverFuncValues[receiverName][funcName] = funcValue

	return nil
}

func (g *Generator) generateFieldsVar(typeName string, fieldNames []struc.FieldName) (string, string) {

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
	return varName, arrayVar
}

func (g *Generator) getFieldArrayType(typeName string) string {
	g.used.fieldArrayType = true
	return arrayType(g.getFieldType(typeName))
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName) bool {
	return g.OnlyExported && isPrivate(fieldName)
}

func (g *Generator) generateTagsVar(typeName string, tagNames []struc.TagName) (string, string) {

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
	return varName, arrayVar
}

func (g *Generator) getTagArrayType(typeName string) string {
	g.used.tagArrayType = true
	return arrayType(g.getTagType(typeName))
}

func (g *Generator) generateGetFieldValueFunc(typeName string, fieldNames []struc.FieldName) (string, string, string) {

	var fieldType string
	if g.WrapType {
		fieldType = g.getFieldType(typeName)
	} else {
		fieldType = baseType
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

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

	return typeName, funcName, funcBody
}

func (g *Generator) generateGetFieldValueByTagValueFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string) {

	var valType string
	if g.WrapType {
		valType = g.getTagValueType(typeName)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

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

	return typeName, funcName, funcBody
}

func (g *Generator) generateGetFieldValuesByTagFuncGeneric(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string) {

	var tagType = baseType
	if g.WrapType {
		tagType = g.getTagType(typeName)
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

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

	return typeName, funcName, funcBody
}

func (g *Generator) generateGetFieldValuesByTagFunctions(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) ([]string, map[string]string, error) {

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	resultType := "[]interface{}"

	funcNames := make([]string, len(tagNames))
	funcBodies := make(map[string]string, len(tagNames))
	for i, tagName := range tagNames {
		funcName := goName("GetFieldValuesByTag"+camel(string(tagName)), g.Export)
		funcBody := "func (" + receiverVar + " *" + typeName + ") " + funcName + "() " + resultType + " " +
			"{\n"

		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, fieldNames, fields)

		funcBody += "return " + fieldExpr + "\n"
		funcBody += "}\n"

		funcNames[i] = funcName
		if _, ok := funcBodies[funcName]; ok {
			return nil, nil, errors.Errorf("duplicated function %s", funcName)
		}
		funcBodies[funcName] = funcBody
	}
	return funcNames, funcBodies, nil
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

func (g *Generator) asRefIfNeed(receiverVar string) string {
	receiverRef := receiverVar
	if g.ReturnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func (g *Generator) generateArrayToExcludesFunc(receiver bool, typeName, arrayTypeName string) (string, string) {
	funcName := goName("Excludes", g.Export)
	receiverVar := "v"
	funcDecl := "func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "(excludes ..." + typeName + ") " + arrayTypeName + " {\n"
	if !receiver {
		receiverVar = "values"
		funcDecl = "func " + funcName + " (" + receiverVar + " " + arrayTypeName + ", excludes ..." + typeName + ") " + arrayTypeName + " {\n"
	}

	funcBody := funcDecl +
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
		"}\n"

	return funcName, funcBody
}

func (g *Generator) generateArrayToStringsFunc(arrayTypeName string, resultType string) (string, string, string) {
	funcName := goName("Strings", g.Export)
	receiverVar := "v"
	funcBody := "" +
		"func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "() []" + resultType + " {\n" +
		"	strings := make([]" + resultType + ", len(v))\n" +
		"	for i, val := range " + receiverVar + " {\n" +
		"		strings[i] = string(val)\n" +
		"		}\n" +
		"		return strings\n" +
		"	}\n"
	return arrayTypeName, funcName, funcBody
}

func (g *Generator) generateAsMapFunc(typeName string, fieldNames []struc.FieldName) (string, string, string) {
	export := g.Export

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

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

	return typeName, funcName, funcBody
}

func (g *Generator) generateAsTagMapFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string) {
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

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

	return typeName, funcName, funcBody
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
	if len(str.Constants) == 0 {
		return nil
	}
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

	for _, constName := range str.Constants {
		text, ok := str.ConstantTemplates[constName]
		if !ok {
			continue
		}
		constName = goName(constName, g.Export)
		if constVal, err := g.generateConst(constName, text, data); err != nil {
			return err
		} else if err = g.addConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateConst(constName string, constTemplate string, data ConstTemplateData) (string, error) {
	add := func(first int, second int) int {
		return first + second
	}
	tmpl, err := template.New(constName).Funcs(template.FuncMap{"add": add}).Parse(constTemplate)
	if err != nil {
		return "", errors.Wrapf(err, "const: %s", constName)
	}

	buf := bytes.Buffer{}
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return splitLines(buf.String(), g.ConstLength)
}

func (g *Generator) filterInjected() {
	g.typeNames = filterNotExisted(g.typeNames, g.typeValues)
	g.constNames = filterNotExisted(g.constNames, g.constValues)
	g.varNames = filterNotExisted(g.varNames, g.varValues)
	g.funcNames = filterNotExisted(g.funcNames, g.funcValues)
	g.funcNames = filterNotExisted(g.funcNames, g.funcValues)

	for _, receiverName := range g.receiverNames {
		recFuncNames, hasFuncs := g.receiverFuncs[receiverName]
		if hasFuncs {
			recFuncValues, hasFuncValues := g.receiverFuncValues[receiverName]
			if hasFuncValues {
				recFuncNames = filterNotExisted(recFuncNames, recFuncValues)
				g.receiverFuncs[receiverName] = recFuncNames
			}
		}
	}
}

func filterNotExisted(names []string, values map[string]string) []string {
	newTypeNames := make([]string, 0)
	var prev *string
	for _, name := range names {
		if len(name) == 0 {
			if prev != nil && len(*prev) > 0 {
				newTypeNames = append(newTypeNames, name)
			}
		} else if _, ok := values[name]; ok {
			newTypeNames = append(newTypeNames, name)
		}
		prev = &name
	}
	return newTypeNames
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
