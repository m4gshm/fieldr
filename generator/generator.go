package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"reflect"
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
	Name string

	IncludedTags []struc.TagName
	FoundTags    []struc.TagName

	Conf    *Config
	Content *ContentConfig

	body              *bytes.Buffer
	used              Used
	excludedTagValues map[string]bool
	Constants         []string

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

const DefaultConstLength = 80

type Config struct {
	Nolint           *bool
	Export           *bool
	NoReceiver       *bool
	ExportVars       *bool
	AllFields        *bool
	ReturnRefs       *bool
	WrapType         *bool
	HardcodeValues   *bool
	NoEmptyTag       *bool
	Compact          *bool
	ConstLength      *int
	ConstReplace     *[]string
	OutBuildTags     *string
	IncludeFieldTags *string
}

type ContentConfig struct {
	Constants *[]string

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

func (g *ContentConfig) IsAll() bool {
	var (
		generateAll = true
		elem        = reflect.ValueOf(g).Elem()
		field       = elem.NumField()
	)
	for i := 0; i < field; i++ {
		structField := elem.Field(i)
		sfk := structField.Kind()
		if sfk == reflect.Ptr {
			elem := structField.Elem()
			noGenerate := isNoGenerate(elem)
			generateAll = generateAll && noGenerate
			if !generateAll {
				break
			}
		}
	}
	return generateAll
}

func isNoGenerate(elem reflect.Value) bool {
	var notGenerate bool
	kind := elem.Kind()
	switch kind {
	case reflect.Bool:
		notGenerate = !elem.Bool()
	case reflect.String:
		s := elem.String()
		notGenerate = len(s) == 0
	case reflect.Slice:
		notGenerate = true
		l := elem.Len()
		for i := 0; i < l; i++ {
			value := elem.Index(i)
			ng := isNoGenerate(value)
			notGenerate = notGenerate && ng
		}
	}
	return notGenerate
}

func (c *Config) MergeWith(src *Config, constantReplacers map[string]string) (*Config, error) {
	copyTrue(src.Nolint, c.Nolint)
	copyTrue(src.Export, c.Export)
	copyTrue(src.ExportVars, c.ExportVars)
	copyTrue(src.AllFields, c.AllFields)
	copyTrue(src.ReturnRefs, c.ReturnRefs)
	copyTrue(src.WrapType, c.WrapType)
	copyTrue(src.HardcodeValues, c.HardcodeValues)
	copyTrue(src.NoEmptyTag, c.NoEmptyTag)
	copyTrue(src.Compact, c.Compact)

	if len(*c.IncludeFieldTags) == 0 && len(*src.IncludeFieldTags) != 0 {
		c.IncludeFieldTags = src.IncludeFieldTags
	}

	if c.ConstLength == nil || *c.ConstLength == DefaultConstLength {
		c.ConstLength = src.ConstLength
	}

	if len(*src.ConstReplace) > 0 {
		newElems, err := struc.ExtractReplacers(*src.ConstReplace...)
		if err != nil {
			return nil, err
		}

		for k, v := range newElems {
			if _, ok := constantReplacers[k]; !ok {
				constantReplacers[k] = v
			}
		}
	}

	if len(*src.OutBuildTags) > 0 && len(*c.OutBuildTags) == 0 {
		c.OutBuildTags = src.OutBuildTags
	}
	return c, nil
}

func copyTrue(s *bool, d *bool) {
	if *s {
		*d = *s
	}
}

func (g *Generator) writeBody(format string, args ...interface{}) {
	if _, err := fmt.Fprintf(g.body, format, args...); err != nil {
		log.Print(fmt.Errorf("writeBody; %w", err))
	}
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

func (g *Generator) GenerateFile(structModel *struc.StructModel, file *ast.File, fileInfo *token.File) error {
	g.excludedTagValues = make(map[string]bool)
	if *g.Conf.NoEmptyTag {
		for fieldName, _tagNames := range structModel.FieldsTagValue {
			for tagName, tagValue := range _tagNames {
				tagValueConstName := g.getTagValueConstName(structModel.TypeName, tagName, fieldName, tagValue)
				if isEmpty(tagValue) {
					g.excludedTagValues[tagValueConstName] = true
				}
			}
		}
	}

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

	if err := g.generateConstants(structModel); err != nil {
		return err
	}

	all := g.Content.IsAll()

	if all || *g.Content.Fields {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldsVar(structModel.TypeName, structModel.FieldNames)); err != nil {
			return err
		}
	}

	if all || *g.Content.Tags {
		g.addVarDelim()
		if err := g.addVar(g.generateTagsVar(structModel.TypeName, structModel.TagNames)); err != nil {
			return err
		}
	}

	if all || *g.Content.FieldTagsMap {
		g.addVarDelim()
		if err := g.addVar(
			g.generateFieldTagsMapVar(structModel.TypeName, structModel.TagNames, structModel.FieldNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || len(*g.Content.TagValues) > 0 {
		if len(structModel.TagNames) == 0 {
			return g.noTagsError("TagValues")
		}

		g.addVarDelim()
		values := *g.Content.TagValues
		if len(*g.Content.TagValues) == 0 {
			values = getTagsValues(structModel.TagNames)
		}
		vars, bodies, err := g.generateTagValuesVar(structModel.TypeName, values, structModel.FieldNames, structModel.FieldsTagValue)
		if err != nil {
			return err
		}
		for _, varName := range vars {
			if err = g.addVar(varName, bodies[varName], nil); err != nil {
				return err
			}
		}
	}

	if all || *g.Content.TagValuesMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagValuesMapVar(structModel.TypeName, structModel.TagNames, structModel.FieldNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || *g.Content.TagFieldsMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagFieldsMapVar(structModel.TypeName, structModel.TagNames, structModel.FieldNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || *g.Content.FieldTagValueMap {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldTagValueMapVar(structModel.FieldNames, structModel.TagNames, structModel.TypeName, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || *g.Content.GetFieldValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueFunc(structModel.TypeName, structModel.FieldNames)); err != nil {
			return err
		}
	}
	if all || *g.Content.GetFieldValueByTagValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueByTagValueFunc(structModel.TypeName, structModel.FieldNames, structModel.TagNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || (*g.Content.GetFieldValuesByTagGeneric) {
		if err := g.addReceiverFunc(g.generateGetFieldValuesByTagFuncGeneric(structModel.TypeName, structModel.FieldNames, structModel.TagNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || len(*g.Content.GetFieldValuesByTag) > 0 {
		funcNames, funcBodies, err := g.generateGetFieldValuesByTagFunctions(structModel.TypeName, structModel.FieldNames, structModel.TagNames, structModel.FieldsTagValue)
		if err != nil {
			return err
		}
		for _, funcName := range funcNames {
			funcBody := funcBodies[funcName]
			if err = g.addReceiverFunc(structModel.TypeName, funcName, funcBody, nil); err != nil {
				return err
			}
		}
	}

	if all || *g.Content.AsMap {
		if err := g.addReceiverFunc(g.generateAsMapFunc(structModel.TypeName, structModel.FieldNames)); err != nil {
			return err
		}
	}
	if all || *g.Content.AsTagMap {
		if err := g.addReceiverFunc(g.generateAsTagMapFunc(structModel.TypeName, structModel.FieldNames, structModel.TagNames, structModel.FieldsTagValue)); err != nil {
			return err
		}
	}

	if err := g.generateHead(structModel.TypeName, structModel.TagNames, structModel.FieldNames, structModel.FieldsTagValue, all); err != nil {
		return err
	}

	rewrite := true
	if file != nil {
		rewrite = false
		for _, comment := range file.Comments {
			pos := comment.Pos()
			base := fileInfo.Base()
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
		g.writeHead(structModel)

		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeReceiverFunctions()
		g.writeFunctions()
	} else {

		//injects

		base := fileInfo.Base()
		chunkVals := make(map[int]map[int]string)

		for _, decl := range file.Decls {
			switch dt := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range dt.Specs {
					switch st := spec.(type) {
					case *ast.TypeSpec:
						if _, ok := st.Type.(*ast.Ident); !ok {
							continue
						}

						start := int(st.Type.Pos()) - base
						end := int(st.Type.End()) - base
						name := st.Name.Name
						if newValue, found := g.typeValues[name]; found {
							chunkVals[start] = map[int]string{end: newValue}
							delete(g.typeValues, name)
						}
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

		name := fileInfo.Name()
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

func (g *Generator) getUsedTags(allTags []struc.TagName) []struc.TagName {
	var usedTags []struc.TagName
	v := *g.Content.GetFieldValuesByTag
	if len(v) > 0 {
		usedTagNames := toSet(v)
		usedTags = make([]struc.TagName, 0, len(usedTagNames))
		for k := range usedTagNames {
			usedTags = append(usedTags, struc.TagName(k))
		}
	} else {
		usedTags = allTags
	}
	return usedTags
}

func toSet(values []string) map[string]interface{} {
	set := make(map[string]interface{})
	for _, value := range values {
		set[value] = nil
	}
	return set
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

func (g *Generator) writeHead(str *struc.StructModel) {
	g.writeBody("// %s'; DO NOT EDIT.\n\n", g.generatedMarker())
	g.writeBody(*g.Conf.OutBuildTags)
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

func (g *Generator) generateHead(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, all bool) error {

	fieldType := baseType
	tagType := baseType
	tagValType := baseType

	usedFieldType := g.used.fieldType
	usedTagType := g.used.tagType
	usedTagValueType := g.used.tagValueType

	if usedFieldType {
		fieldType = getFieldType(typeName, *g.Conf.Export)
	}
	if usedTagType {
		tagType = getTagType(typeName, *g.Conf.Export)
	}
	if usedTagValueType {
		tagValType = getTagValueType(typeName, *g.Conf.Export)
	}

	if *g.Conf.WrapType {
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

	fieldConstName := g.used.fieldConstName || *g.Content.EnumFields || all
	tagConstName := g.used.tagConstName || *g.Content.EnumTags || all
	tagValueConstName := g.used.tagValueConstName || *g.Content.EnumTagValues || all

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

	if *g.Conf.WrapType {
		if all || *g.Content.Strings {
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

		if *g.Content.Excludes {
			if g.used.fieldArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, fieldType, arrayType(fieldType))
				if err := g.addReceiverFunc(fieldType, funcName, funcBody, nil); err != nil {
					return err
				}
			}

			if g.used.tagArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagType, arrayType(tagType))
				if err := g.addReceiverFunc(tagType, funcName, funcBody, nil); err != nil {
					return err
				}
			}

			if g.used.tagValueArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagValType, arrayType(tagValType))
				if err := g.addReceiverFunc(tagValType, funcName, funcBody, nil); err != nil {
					return err
				}
			}
		}
	} else {
		if *g.Content.Excludes {
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
	return getFieldType(typeName, *g.Conf.Export)
}

func (g *Generator) getTagType(typeName string) string {
	g.used.tagType = true
	return getTagType(typeName, *g.Conf.Export)
}

func (g *Generator) getTagValueType(typeName string) string {
	g.used.tagValueType = true
	return getTagValueType(typeName, *g.Conf.Export)
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

func (g *Generator) generateFieldTagValueMapVar(fieldNames []struc.FieldName, tagNames []struc.TagName, typeName string, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, error) {
	varName := goName(typeName+"_FieldTagValue", *g.Conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	var varValue string
	fieldType := baseType
	tagType := baseType
	tagValueType := baseType
	if *g.Conf.WrapType {
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

		compact := *g.Conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
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

	return varName, varValue, nil
}

func (g *Generator) generateFieldTagsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, error) {
	varName := goName(typeName+"_FieldTags", *g.Conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	fieldType := baseType
	tagArrayType := "[]" + baseType

	if *g.Conf.WrapType {
		tagArrayType = g.getTagArrayType(typeName)
		fieldType = g.getFieldType(typeName)
	}

	varValue := "map[" + fieldType + "]" + tagArrayType + "{\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		fieldConstName := g.getFieldConstName(typeName, fieldName)

		if *g.Conf.WrapType {
			varValue += fieldConstName + ": " + tagArrayType + "{"
		} else {
			varValue += fieldConstName + ": []" + baseType + "{"
		}

		compact := *g.Conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
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

	return varName, varValue, nil
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
	if *g.Conf.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
	}

	for _, tagName := range tagNames {
		varName := goName(typeName+"_TagValues_"+string(tagName), *g.Conf.ExportVars)
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

func (g *Generator) generateTagValuesMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, error) {
	varName := goName(typeName+"_TagValues", *g.Conf.ExportVars)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := baseType
	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType

	if *g.Conf.WrapType {
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

	return varName, varValue, nil
}

func (g *Generator) generateTagValueBody(typeName string, tagValueArrayType string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagName struc.TagName) string {
	var varValue string
	if *g.Conf.WrapType {
		varValue += tagValueArrayType + "{"
	} else {
		varValue += "[]" + baseType + "{"
	}

	compact := *g.Conf.Compact || g.generatedAmount(fieldNames) <= oneLineSize
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

func (g *Generator) generateTagFieldsMapVar(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, error) {
	varName := goName(typeName+"_TagFields", *g.Conf.ExportVars)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := baseType
	fieldArrayType := "[]" + baseType

	if *g.Conf.WrapType {
		tagType = g.getTagType(typeName)
		fieldArrayType = g.getFieldArrayType(typeName)
	}

	varValue := "map[" + tagType + "]" + fieldArrayType + "{\n"

	for _, tagName := range tagNames {
		constName := g.getTagConstName(typeName, tagName)

		varValue += constName + ": " + fieldArrayType + "{"

		compact := *g.Conf.Compact || g.generatedAmount(fieldNames) <= oneLineSize
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
	return varName, varValue, nil
}

func (g *Generator) generateTagFieldConstants(typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagValueType string) error {
	if len(tagNames) == 0 {
		return g.noTagsError("Tag Fields Constants")
	}

	g.addConstDelim()
	for _, _tagName := range tagNames {
		for _, _fieldName := range fieldNames {
			_tagValue, ok := fields[_fieldName][_tagName]
			if ok {
				isEmptyTag := isEmpty(_tagValue)

				if isEmptyTag {
					_tagValue = struc.TagValue(_fieldName)
				}

				tagValueConstName := getTagValueConstName(typeName, _tagName, _fieldName, *g.Conf.Export)
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
		constName := getFieldConstName(typeName, fieldName, *g.Conf.Export)
		constVal := g.getConstValue(fieldType, string(fieldName))
		if err := g.addConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateTagConstants(typeName string, tagType string, tagNames []struc.TagName) error {
	if len(tagNames) == 0 {
		return g.noTagsError("Tag Constants")
	}
	g.addConstDelim()
	for _, name := range tagNames {
		constName := getTagConstName(typeName, name, *g.Conf.Export)
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
	if *g.Conf.WrapType {
		return fmt.Sprintf("%v(\"%v\")", typ, value)
	}
	return fmt.Sprintf("\"%v\"", value)
}

func (g *Generator) addVarDelim() {
	if len(g.varNames) > 0 {
		g.varNames = append(g.varNames, "")
	}
}

func (g *Generator) addVar(varName, varValue string, err error) error {
	if err != nil {
		return err
	}
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

func (g *Generator) addReceiverFunc(receiverName, funcName, funcValue string, err error) error {
	if err != nil {
		return err
	}
	functions, ok := g.receiverFuncs[receiverName]
	if !ok {
		g.receiverNames = append(g.receiverNames, receiverName)

		functions = make([]string, 0)
		g.receiverFuncs[receiverName] = functions
		g.receiverFuncValues[receiverName] = make(map[string]string)
	}

	if _, ok = g.receiverFuncValues[receiverName][funcName]; ok {
		return errors.Errorf("duplicated receiver's func %v.%v", receiverName, funcName)
	}

	g.receiverFuncs[receiverName] = append(functions, funcName)
	g.receiverFuncValues[receiverName][funcName] = funcValue

	return nil
}

func (g *Generator) generateFieldsVar(typeName string, fieldNames []struc.FieldName) (string, string, error) {

	var arrayVar string
	if *g.Conf.WrapType {
		arrayVar = g.getFieldArrayType(typeName) + "{"
	} else {
		arrayVar = "[]" + baseType + "{"
	}

	compact := *g.Conf.Compact || g.generatedAmount(fieldNames) <= oneLineSize
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
	varName := goName(varNameTemplate, *g.Conf.ExportVars)
	return varName, arrayVar, nil
}

func (g *Generator) getFieldArrayType(typeName string) string {
	g.used.fieldArrayType = true
	return arrayType(g.getFieldType(typeName))
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName) bool {
	return !*g.Conf.AllFields && isPrivate(fieldName)
}

func (g *Generator) generateTagsVar(typeName string, tagNames []struc.TagName) (string, string, error) {
	varName := goName(typeName+"_Tags", *g.Conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagArrayType := "[]" + baseType

	if *g.Conf.WrapType {
		tagArrayType = g.getTagArrayType(typeName)
	}

	arrayVar := tagArrayType + "{"

	compact := *g.Conf.Compact || len(tagNames) <= oneLineSize

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

	return varName, arrayVar, nil
}

func (g *Generator) getTagArrayType(typeName string) string {
	g.used.tagArrayType = true
	return arrayType(g.getTagType(typeName))
}

func (g *Generator) generateGetFieldValueFunc(typeName string, fieldNames []struc.FieldName) (string, string, string, error) {

	var fieldType string
	if *g.Conf.WrapType {
		fieldType = g.getFieldType(typeName)
	} else {
		fieldType = baseType
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	funcName := goName("GetFieldValue", *g.Conf.Export)
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ", " + valVar + " " + fieldType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + fieldType + ") interface{}"
	}
	funcBody += " {" + g.noLint() + "\n" + "switch " + valVar + " {\n"

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

	return typeName, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValueByTagValueFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string, error) {
	funcName := goName("GetFieldValueByTagValue", *g.Conf.Export)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}
	var valType string
	if *g.Conf.WrapType {
		valType = g.getTagValueType(typeName)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ", " + valVar + " " + valType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + valType + ") interface{}"
	}
	funcBody += " {" + g.noLint() + "\n"
	funcBody += "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		var caseExpr string

		compact := *g.Conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
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

	return typeName, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFuncGeneric(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fieldTagValues map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string, error) {
	funcName := goName("GetFieldValuesByTag", *g.Conf.Export)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	var tagType = baseType
	if *g.Conf.WrapType {
		tagType = g.getTagType(typeName)
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	resultType := "[]interface{}"
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ", " + valVar + " " + tagType + ") " + resultType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + valVar + " " + tagType + ") " + resultType
	}
	funcBody += " {" + g.noLint() + "\n" + "switch " + valVar + " {\n"

	for _, tagName := range tagNames {
		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, fieldNames, fieldTagValues)

		caseExpr := g.getTagConstName(typeName, tagName)
		funcBody += "case " + caseExpr + ":\n" +
			"return " + fieldExpr + "\n"

	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeName, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFunctions(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) ([]string, map[string]string, error) {

	usedTags := g.getUsedTags(tagNames)

	const funcNamePrefix = "GetFieldValuesByTag"
	if len(tagNames) == 0 {
		msg := ""
		for _, tagName := range usedTags {
			if len(msg) > 0 {
				msg += ","
			}
			msg += g.getFuncName(funcNamePrefix, tagName)
		}
		return nil, nil, g.noTagsError(msg)
	}

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	resultType := "[]interface{}"

	funcNames := make([]string, len(usedTags))
	funcBodies := make(map[string]string, len(usedTags))
	for i, tagName := range usedTags {
		funcName := g.getFuncName(funcNamePrefix, tagName)
		var funcBody string
		if *g.Conf.NoReceiver {
			funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ") " + resultType
		} else {
			funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "() " + resultType
		}
		funcBody += " {" + g.noLint() + "\n"

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

func (g *Generator) getFuncName(funcNamePrefix string, tagName struc.TagName) string {
	return goName(funcNamePrefix+camel(string(tagName)), *g.Conf.Export)
}

func (g *Generator) fieldValuesArrayByTag(receiverRef string, resultType string, tagName struc.TagName, fieldNames []struc.FieldName, tagFieldValues map[struc.FieldName]map[struc.TagName]struc.TagValue) string {
	fieldExpr := ""

	compact := *g.Conf.Compact || g.generatedAmount(fieldNames) <= oneLineSize
	if !compact {
		fieldExpr += "\n"
	}

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		_, ok := tagFieldValues[fieldName][tagName]
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
	if *g.Conf.ReturnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func (g *Generator) generateArrayToExcludesFunc(receiver bool, typeName, arrayTypeName string) (string, string) {
	funcName := goName("Excludes", *g.Conf.Export)
	receiverVar := "v"
	funcDecl := "func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "(excludes ..." + typeName + ") " + arrayTypeName + " {" + g.noLint() + "\n"
	if !receiver {
		receiverVar = "values"
		funcDecl = "func " + funcName + " (" + receiverVar + " " + arrayTypeName + ", excludes ..." + typeName + ") " + arrayTypeName + " {" + g.noLint() + "\n"
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

func (g *Generator) generateArrayToStringsFunc(arrayTypeName string, resultType string) (string, string, string, error) {
	funcName := goName("Strings", *g.Conf.Export)
	receiverVar := "v"
	funcBody := "" +
		"func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "() []" + resultType + " {" + g.noLint() + "\n" +
		"	strings := make([]" + resultType + ", len(v))\n" +
		"	for i, val := range " + receiverVar + " {\n" +
		"		strings[i] = string(val)\n" +
		"		}\n" +
		"		return strings\n" +
		"	}\n"
	return arrayTypeName, funcName, funcBody, nil
}

func (g *Generator) generateAsMapFunc(typeName string, fieldNames []struc.FieldName) (string, string, string, error) {
	export := *g.Conf.Export

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	keyType := baseType
	if *g.Conf.WrapType {
		keyType = g.getFieldType(typeName)
	}

	funcName := goName("AsMap", export)
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ") map[" + keyType + "]interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "() map[" + keyType + "]interface{}"
	}
	funcBody += " {" + g.noLint() + "\n" +
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

	return typeName, funcName, funcBody, nil
}

func (g *Generator) generateAsTagMapFunc(typeName string, fieldNames []struc.FieldName, tagNames []struc.TagName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue) (string, string, string, error) {
	funcName := goName("AsTagMap", *g.Conf.Export)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	tagValueType := baseType
	tagType := baseType
	if *g.Conf.WrapType {
		tagValueType = g.getTagValueType(typeName)
		tagType = g.getTagType(typeName)
	}

	valueType := "interface{}"

	varName := "tag"

	mapType := "map[" + tagValueType + "]" + valueType

	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeName + ", " + varName + " " + tagType + ") " + mapType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeName + ") " + funcName + "(" + varName + " " + tagType + ") " + mapType
	}

	funcBody += " {" + g.noLint() + "\n" +
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

	return typeName, funcName, funcBody, nil
}

func (g *Generator) noTagsError(funcName string) error {
	includedTags := g.IncludedTags
	if len(includedTags) > 0 {
		return errors.Errorf(funcName+"; no tags for generating; included: %v", includedTags)
	} else {
		return errors.Errorf(funcName + "; no tags for generating;")
	}
}

func (g *Generator) getTagConstName(typeName string, tag struc.TagName) string {
	if *g.Conf.HardcodeValues {
		return quoted(tag)
	}
	g.used.tagConstName = true
	return getTagConstName(typeName, tag, *g.Conf.Export)
}

func getTagConstName(typeName string, tag struc.TagName, export bool) string {
	return goName(getTagType(typeName, export)+"_"+string(tag), export)
}

func (g *Generator) getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, tagVal struc.TagValue) string {
	if *g.Conf.HardcodeValues {
		return quoted(tagVal)
	}
	g.used.tagValueConstName = true
	export := isExport(fieldName, *g.Conf.Export)
	return getTagValueConstName(typeName, tag, fieldName, export)
}

func getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, export bool) string {
	export = isExport(fieldName, export)
	return goName(getTagValueType(typeName, export)+"_"+string(tag)+"_"+string(fieldName), export)
}

func (g *Generator) getFieldConstName(typeName string, fieldName struc.FieldName) string {
	if *g.Conf.HardcodeValues {
		return quoted(fieldName)
	}
	g.used.fieldConstName = true
	return getFieldConstName(typeName, fieldName, isExport(fieldName, *g.Conf.Export))
}

type ConstTemplateData struct {
	Fields        []string
	Tags          []string
	FieldTags     map[string][]string
	TagValues     map[string][]string
	TagFields     map[string][]string
	FieldTagValue map[string]map[string]string
}

func (g *Generator) generateConstants(str *struc.StructModel) error {
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
		constName = goName(constName, *g.Conf.Export)
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

	return g.splitLines(buf.String())
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

func (g *Generator) noLint() string {
	if *g.Conf.Nolint {
		return "//nolint"
	}
	return ""
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

func (g *Generator) splitLines(generatedValue string) (string, error) {
	stepSize := *g.Conf.ConstLength
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

		line := 1
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
							buf.WriteString(" + ")
							if line == 1 {
								buf.WriteString(g.noLint())
							}
							buf.WriteString("\n")
							buf.WriteString(quotes)
							line++
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
							if line == 1 {
								buf.WriteString(g.noLint())
							}
							buf.WriteString("\n")
							line++
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
