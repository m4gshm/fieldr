package generator

import (
	"bytes"
	"fmt"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/go/packages"
)

const oneLineSize = 3

type TransformTrigger string

const (
	TransformTriggerEmpty TransformTrigger = ""
	TransformTriggerField TransformTrigger = "field"
	TransformTriggerType  TransformTrigger = "type"
)

type TransformEngine string

const (
	TransformEngineFmt TransformEngine = "fmt"
)

type Generator struct {
	Name string

	IncludedTags []struc.TagName
	FoundTags    []struc.TagName

	Conf    *Config
	Content *ContentConfig

	body *bytes.Buffer
	used Used

	excludedTagValues          map[string]bool
	excludedFields             map[struc.FieldName]interface{}
	transformValuesByFieldName map[struc.FieldName][]func(string) string
	transformValuesByFieldType map[struc.FieldType][]func(string) string
	transformValues            []func(string) string

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
	Nolint              *bool
	Export              *bool
	NoReceiver          *bool
	ExportVars          *bool
	AllFields           *bool
	ReturnRefs          *bool
	WrapType            *bool
	HardcodeValues      *bool
	NoEmptyTag          *bool
	Compact             *bool
	Snake               *bool
	Flat                *[]string
	ConstLength         *int
	ConstReplace        *[]string
	OutBuildTags        *string
	IncludeFieldTags    *string
	OutPackage          *string
	Name                *string
	ExcludeFields       *[]string
	TransformFieldValue *[]string
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
	copyTrue(src.Snake, c.Snake)

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

	if len(*src.Name) > 0 && len(*c.Name) == 0 {
		c.Name = src.Name
	}

	if len(*c.ExcludeFields) == 0 && len(*src.ExcludeFields) != 0 {
		c.ExcludeFields = src.ExcludeFields
	}

	if len(*c.TransformFieldValue) == 0 && len(*src.TransformFieldValue) != 0 {
		c.TransformFieldValue = src.TransformFieldValue
	}

	if len(*c.Flat) == 0 && len(*src.Flat) != 0 {
		c.Flat = src.Flat
	}

	return c, nil
}

func copyTrue(s *bool, d *bool) {
	if s != nil && *s {
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

func (g *Generator) GenerateFile(model *struc.Model, outFile *ast.File, outFileInfo *token.File, outPackage *packages.Package) error {
	outPackagePath := outPackage.PkgPath
	var outPackageName string
	ref := g.Conf.OutPackage
	if ref != nil && len(*ref) > 0 {
		outPackageName = *ref
	} else {
		name := outPackage.Name
		if len(name) > 0 {
			outPackageName = name
		} else {
			outPackageName = packagePathToName(outPackagePath)
		}
		logger.Debugw("output package %v, path %v", outPackageName, outPackagePath)
	}

	needImport := model.PackagePath != outPackagePath

	structPackage := ""
	if needImport {
		structPackage = model.PackageName
	}

	isRewrite := g.isRewrite(outFile, outFileInfo)
	if !isRewrite && needImport {
		alias, found, err := g.findImportPackageAlias(model, outFile)
		if err != nil {
			return err
		}
		if found {
			needImport = false
			if len(alias) > 0 {
				structPackage = alias
			}
		} else {
			structPackageSuffixed := structPackage
			duplicated := false
			i := 0
			for i <= 100 {
				if duplicated, err = g.hasDuplicatedPackage(outFile, structPackageSuffixed); err != nil {
					return err
				} else if duplicated {
					i++
					structPackageSuffixed = structPackage + strconv.Itoa(i)
				} else {
					break
				}
			}
			if !duplicated && i > 0 {
				structPackage = structPackageSuffixed
			}
		}
	}

	g.excludedTagValues = make(map[string]bool)
	if *g.Conf.NoEmptyTag {
		for fieldName, _tagNames := range model.FieldsTagValue {
			for tagName, tagValue := range _tagNames {
				tagValueConstName := g.getUsedTagValueConstName(model.TypeName, tagName, fieldName, tagValue)
				if isEmpty(tagValue) {
					g.excludedTagValues[tagValueConstName] = true
				}
			}
		}
	}

	excludedFields := make(map[struc.FieldName]interface{})
	for _, excludes := range *g.Conf.ExcludeFields {
		e := strings.Split(excludes, struc.ListValuesSeparator)
		for _, exclude := range e {
			excludedFields[exclude] = nil
		}
	}

	g.excludedFields = make(map[struc.FieldName]interface{})
	for _, fieldName := range model.FieldNames {
		if _, excluded := excludedFields[fieldName]; excluded {
			g.excludedFields[fieldName] = nil
		}
	}

	var transformValues []func(string) string
	transformValuesByFieldName := map[struc.FieldName][]func(string) string{}
	transformValuesByFieldType := map[struc.FieldType][]func(string) string{}
	for _, transformValue := range *g.Conf.TransformFieldValue {
		transforms := strings.Split(transformValue, struc.ListValuesSeparator)
		for _, transform := range transforms {
			transformParts := strings.Split(transform, struc.KeyValueSeparator)
			var transformTrigger TransformTrigger
			var transformTriggerValue string
			var transformer string
			if len(transformParts) == 1 {
				transformTrigger = TransformTriggerEmpty
				transformTriggerValue = transformParts[0]
				transformer = transformTriggerValue
			} else if len(transformParts) == 2 {
				transformTrigger = TransformTriggerField
				transformTriggerValue = transformParts[0]
				transformer = transformParts[1]
			} else if len(transformParts) == 3 {
				transformTrigger = TransformTrigger(transformParts[0])
				transformTriggerValue = transformParts[1]
				transformer = transformParts[2]
			} else {
				return errors.Errorf("Unsupported transformValue format '%v'", transform)
			}

			var transformerEngine TransformEngine
			var transformerEngineData string
			transformerParts := strings.Split(transformer, struc.ReplaceableValueSeparator)
			if len(transformerParts) == 0 {
				return errors.Errorf("Undefined transformer value '%v'", transform)
			} else if len(transformerParts) == 2 {
				transformerEngine = TransformEngine(transformerParts[0])
				transformerEngineData = transformerParts[1]
			} else {
				return errors.Errorf("Unsupported transformer value '%v' from '%v'", transformerParts[0], transformer)
			}

			var transformFunc func(string) string
			switch transformerEngine {
			case TransformEngineFmt:
				transformFunc = func(fieldValue string) string {
					return fmt.Sprintf(transformerEngineData, fieldValue)
				}
			default:
				return errors.Errorf("Unsupported transform engine '%v' from '%v'", transformerEngine, transform)
			}

			switch transformTrigger {
			case TransformTriggerEmpty:
				transformValues = append(transformValues, transformFunc)
			case TransformTriggerField:
				fieldName := transformTriggerValue
				fieldNameTransformEngines, ok := transformValuesByFieldName[fieldName]
				if !ok {
					fieldNameTransformEngines = []func(string) string{}
				}
				transformValuesByFieldName[fieldName] = append(fieldNameTransformEngines, transformFunc)
			case TransformTriggerType:
				fieldType := transformTriggerValue
				fieldNameTransformEngines, ok := transformValuesByFieldType[fieldType]
				if !ok {
					fieldNameTransformEngines = []func(string) string{}
				}
				transformValuesByFieldType[fieldType] = append(fieldNameTransformEngines, transformFunc)
			default:
				return errors.Errorf("Unsupported transform trigger '%v' from '%v'", transformTrigger, transform)
			}
		}
	}
	g.transformValues = transformValues
	g.transformValuesByFieldName = transformValuesByFieldName
	g.transformValuesByFieldType = transformValuesByFieldType

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

	if err := g.generateConstants(model); err != nil {
		return err
	}

	all := g.Content.IsAll()

	if all || *g.Content.Fields {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldsVar(model, model.FieldNames)); err != nil {
			return err
		}
	}

	if all || *g.Content.Tags {
		g.addVarDelim()
		if err := g.addVar(g.generateTagsVar(model.TypeName, model.TagNames)); err != nil {
			return err
		}
	}

	if all || *g.Content.FieldTagsMap {
		g.addVarDelim()
		if err := g.addVar(
			g.generateFieldTagsMapVar(model.TypeName, model.TagNames, model.FieldNames, model.FieldsTagValue)); err != nil {
			return err
		}
	}

	if all || len(*g.Content.TagValues) > 0 {
		if len(model.TagNames) == 0 {
			return g.noTagsError("TagValues")
		}

		g.addVarDelim()
		values := *g.Content.TagValues
		if len(*g.Content.TagValues) == 0 {
			values = getTagsValues(model.TagNames)
		}
		vars, bodies, err := g.generateTagValuesVar(model.TypeName, values, model.FieldNames, model.FieldsTagValue)
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
		if err := g.addVar(g.generateTagValuesMapVar(model)); err != nil {
			return err
		}
	}

	if all || *g.Content.TagFieldsMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagFieldsMapVar(model)); err != nil {
			return err
		}
	}

	if all || *g.Content.FieldTagValueMap {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldTagValueMapVar(model)); err != nil {
			return err
		}
	}

	getFieldValue := *g.Content.GetFieldValue
	getFieldValueByTagValue := *g.Content.GetFieldValueByTagValue
	getFieldValuesByTagGeneric := *g.Content.GetFieldValuesByTagGeneric
	getFieldValuesByTag := *g.Content.GetFieldValuesByTag
	asMap := *g.Content.AsMap
	asTagMap := *g.Content.AsTagMap

	generateManyFuncs := all || (toInt(getFieldValue)+toInt(getFieldValueByTagValue)+
		toInt(getFieldValuesByTagGeneric)+len(getFieldValuesByTag)+toInt(asMap)+toInt(asTagMap)) > 1
	if len(*g.Conf.Name) > 0 && generateManyFuncs {
		return errors.New("-name not supported for multiple functions, please specify only one function")
	}

	if all || getFieldValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueFunc(model, structPackage)); err != nil {
			return err
		}
	}
	if all || getFieldValueByTagValue {
		if err := g.addReceiverFunc(g.generateGetFieldValueByTagValueFunc(model, structPackage)); err != nil {
			return err
		}
	}

	if all || getFieldValuesByTagGeneric {
		if err := g.addReceiverFunc(g.generateGetFieldValuesByTagFuncGeneric(model, structPackage)); err != nil {
			return err
		}
	}

	if all || len(getFieldValuesByTag) > 0 {
		receiverType, funcNames, funcBodies, err := g.generateGetFieldValuesByTagFunctions(model, structPackage)
		if err != nil {
			return err
		}
		for _, funcName := range funcNames {
			funcBody := funcBodies[funcName]
			if err = g.addReceiverFunc(receiverType, funcName, funcBody, nil); err != nil {
				return err
			}
		}
	}

	if all || asMap {
		typeLink, funcName, funcBody, err := g.generateAsMapFunc(model, structPackage)
		if err = g.addReceiverFunc(typeLink, funcName, funcBody, err); err != nil {
			return err
		}
	}
	if all || asTagMap {
		if err := g.addReceiverFunc(g.generateAsTagMapFunc(model, structPackage)); err != nil {
			return err
		}
	}

	if err := g.generateHead(model, all); err != nil {
		return err
	}

	if isRewrite {
		g.body = &bytes.Buffer{}
		g.writeHead(model, outPackageName, needImport)

		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeReceiverFunctions()
		g.writeFunctions()
	} else {
		//injects
		chunks, err := g.getInjectChunks(model, outFile, outFileInfo.Base(), needImport, structPackage)
		if err != nil {
			return err
		}
		name := outFileInfo.Name()
		fileBytes, err := ioutil.ReadFile(name)
		if err != nil {
			return err
		}

		newFileContent := inject(chunks, string(fileBytes))
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

func toInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (g *Generator) isRewrite(outFile *ast.File, outFileInfo *token.File) bool {
	if outFile == nil {
		return true
	}
	for _, comment := range outFile.Comments {
		pos := comment.Pos()
		base := outFileInfo.Base()
		firstComment := int(pos) == base
		if firstComment {
			text := comment.Text()
			generatedMarker := g.generatedMarker()
			generated := strings.HasPrefix(text, generatedMarker)
			return generated
		}
	}
	return false
}

func (g *Generator) findImportPackageAlias(model *struc.Model, outFile *ast.File) (string, bool, error) {
	for _, decl := range outFile.Decls {
		switch dt := decl.(type) {
		case *ast.GenDecl:
			if dt.Tok != token.IMPORT {
				continue
			}
			for _, spec := range dt.Specs {
				switch st := spec.(type) {
				case *ast.ImportSpec:
					if value, err := strconv.Unquote(st.Path.Value); err != nil {
						return "", false, err
					} else if imported := value == model.PackagePath; imported {
						if st.Name != nil {
							return st.Name.Name, imported, nil
						}
						return "", imported, nil
					}
				}
			}
		}
	}
	return "", false, nil
}

func (g *Generator) hasDuplicatedPackage(outFile *ast.File, packageName string) (bool, error) {
	for _, decl := range outFile.Decls {
		switch dt := decl.(type) {
		case *ast.GenDecl:
			if dt.Tok != token.IMPORT {
				continue
			}
			for _, spec := range dt.Specs {
				switch st := spec.(type) {
				case *ast.ImportSpec:
					var name string
					if st.Name != nil {
						name = st.Name.Name
					} else if pathValue, err := strconv.Unquote(st.Path.Value); err != nil {
						return false, err
					} else {
						name = packagePathToName(pathValue)
					}
					if name == packageName {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func (g *Generator) getInjectChunks(model *struc.Model, outFile *ast.File, base int, needImport bool, structPackage string) (map[int]map[int]string, error) {
	noReceiver := g.Conf.NoReceiver != nil && *g.Conf.NoReceiver
	chunks := make(map[int]map[int]string)

	importInjected := false
	for _, decl := range outFile.Decls {
		switch dt := decl.(type) {
		case *ast.GenDecl:
			if needImport && dt.Tok == token.IMPORT {
				expr := g.importExpr(model, structPackage, false)
				var start int
				var end int
				if len(dt.Specs) == 0 {
					start = int(dt.Pos()) - base
					end = int(dt.End()) - base
				} else {
					if dt.Rparen != token.NoPos {
						start = int(dt.Rparen) - base
						end = start
						expr = "\n" + g.importExpr(model, structPackage, true)
					}
				}
				chunks[start] = map[int]string{end: expr}
				importInjected = true
			} else {
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
							chunks[start] = map[int]string{end: newValue}
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
								chunks[start] = map[int]string{end: newValue}
								delete(generatingValues, name)
							}
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
				if _, err := g.addReceiveFuncOnRewrite(recv.List, name, chunks, start, end); err != nil {
					return nil, err
				}
			} else {
				handled := false
				if noReceiver {
					params := dt.Type.Params
					var err error
					handled, err = g.addReceiveFuncOnRewrite(params.List, name, chunks, start, end)
					if err != nil {
						return nil, err
					}
				}
				if !handled {
					funcDecl, hasFuncDecl := g.funcValues[name]
					if hasFuncDecl {
						chunks[start] = map[int]string{end: funcDecl}
						delete(g.funcValues, name)
					}
				}
			}
		}
	}

	if needImport && !importInjected {
		start := int(outFile.Name.End()) - base
		end := start
		chunks[start] = map[int]string{end: g.importExpr(model, structPackage, false)}
	}
	return chunks, nil
}

func (g *Generator) importExpr(model *struc.Model, packageName string, forMultiline bool) string {
	path := model.PackagePath
	name := packagePathToName(path)
	quoted := "\"" + path + "\"\n"
	if name != packageName {
		quoted = packageName + " " + quoted
	}
	if forMultiline {
		return "\n" + quoted
	}
	return "\nimport " + quoted
}

func (g *Generator) addReceiveFuncOnRewrite(list []*ast.Field, name string, chunks map[int]map[int]string, start int, end int) (bool, error) {
	if len(list) == 0 {
		return false, nil
	}
	field := list[0]
	typ := field.Type
	receiverName, err := getReceiverName(typ)
	if err != nil {
		return false, fmt.Errorf("func %v; %w", name, err)
	}
	recFuncs, hasFuncs := g.receiverFuncValues[receiverName]
	if hasFuncs {
		funcDecl, hasFuncDecl := recFuncs[name]
		if hasFuncDecl {
			chunks[start] = map[int]string{end: funcDecl}
			delete(recFuncs, name)
			return true, nil
		}
	}
	return false, nil
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

func getReceiverName(typ ast.Expr) (string, error) {
	switch tt := typ.(type) {
	case *ast.StarExpr:
		return getReceiverName(tt.X)
	case *ast.Ident:
		return tt.Name, nil
	case *ast.SelectorExpr:
		name := tt.Sel.Name
		x := tt.X
		if x != nil {
			pkgAlias := ""
			switch xt := x.(type) {
			case *ast.Ident:
				pkgAlias = xt.Name
			default:
				return "", errors.Errorf("receiver type; unexpected type %v, value %v", reflect.TypeOf(tt), tt)
			}
			if len(pkgAlias) > 0 {
				return pkgAlias + "." + name, nil
			}
		}
		return name, nil
	default:
		return "", errors.Errorf("receiver name; unexpecte type %v, value %v", reflect.TypeOf(tt), tt)
	}
}

func inject(chunks map[int]map[int]string, fileContent string) string {
	sortedPos := getSortedChunks(chunks)
	newFileContent := ""
	start := 0
	for _, end := range sortedPos {
		for j, value := range chunks[end] {
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

func (g *Generator) writeHead(str *struc.Model, packageName string, needImport bool) {
	g.writeBody("// %s'; DO NOT EDIT.\n\n", g.generatedMarker())
	g.writeBody(*g.Conf.OutBuildTags)
	g.writeBody("package %s\n", packageName)

	if needImport {
		g.writeBody(g.importExpr(str, "", false))
	}
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

func (g *Generator) generateHead(model *struc.Model, all bool) error {
	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames

		fieldType  = baseType
		tagType    = baseType
		tagValType = baseType

		usedFieldType    = g.used.fieldType || *g.Content.EnumFields
		usedTagType      = g.used.tagType || *g.Content.EnumTags
		usedTagValueType = g.used.tagValueType || *g.Content.EnumTagValues
	)

	if usedFieldType {
		fieldType = g.getFieldType(typeName)
	}
	if usedTagType {
		tagType = g.getTagType(typeName)
	}
	if usedTagValueType {
		tagValType = g.getTagValueType(typeName)
	}

	wrapType := *g.Conf.WrapType
	if wrapType {
		if usedFieldType || *g.Content.EnumFields {
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
		if err := g.generateFieldConstants(model, fieldType, fieldNames); err != nil {
			return err
		}
	}

	if tagConstName {
		if err := g.generateTagConstants(typeName, tagType, tagNames); err != nil {
			return err
		}
	}

	if tagValueConstName {
		if err := g.generateTagFieldConstants(model, tagValType); err != nil {
			return err
		}
	}

	if wrapType {
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

func (g *Generator) getUsedFieldType(typeName string) string {
	g.used.fieldType = true
	return g.getFieldType(typeName)
}

func (g *Generator) getUsedTagType(typeName string) string {
	g.used.tagType = true
	return g.getTagType(typeName)
}

func (g *Generator) getUsedTagValueType(typeName string) string {
	g.used.tagValueType = true
	return g.getTagValueType(typeName)
}

func arrayType(baseType string) string {
	return baseType + "List"
}

func (g *Generator) getTagValueType(typeName string) string {
	return goName(typeName+g.getIdentPart("TagValue"), *g.Conf.Export)
}

func (g *Generator) getTagType(typeName string) string {
	return goName(typeName+g.getIdentPart("Tag"), *g.Conf.Export)
}

func (g *Generator) getFieldType(typeName string) string {
	return goName(typeName+g.getIdentPart("Field"), *g.Conf.Export)
}

func (g *Generator) getIdentPart(suffix string) string {
	if *g.Conf.Snake {
		return "_" + suffix
	}
	return camel(suffix)
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

func (g *Generator) generateFieldTagValueMapVar(model *struc.Model) (string, string, error) {
	var (
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
		typeName   = model.TypeName
	)

	varName := goName(typeName+g.getIdentPart("FieldTagValue"), *g.Conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	var varValue string
	fieldType := baseType
	tagType := baseType
	tagValueType := baseType
	if *g.Conf.WrapType {
		tagType = g.getUsedTagType(typeName)
		fieldType = g.getUsedFieldType(typeName)
		tagValueType = g.getUsedTagValueType(typeName)
	}
	varValue = "map[" + fieldType + "]map[" + tagType + "]" + tagValueType + "{\n"
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		fieldConstName := g.getUsedFieldConstName(typeName, fieldName)

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

			tagConstName := g.getUsedTagConstName(typeName, tagName)
			tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal)
			if _, excluded := g.excludedTagValues[tagValueConstName]; excluded {
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
	varName := goName(typeName+g.getIdentPart("FieldTags"), *g.Conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	fieldType := baseType
	tagArrayType := "[]" + baseType

	if *g.Conf.WrapType {
		tagArrayType = g.getTagArrayType(typeName)
		fieldType = g.getUsedFieldType(typeName)
	}

	varValue := "map[" + fieldType + "]" + tagArrayType + "{\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		fieldConstName := g.getUsedFieldConstName(typeName, fieldName)

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
			tagConstName := g.getUsedTagConstName(typeName, tagName)
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
		tagValueType = g.getUsedTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
	}

	for _, tagName := range tagNames {
		varName := goName(typeName+g.getIdentPart("TagValues")+g.getIdentPart(string(tagName)), *g.Conf.ExportVars)
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

func (g *Generator) generateTagValuesMapVar(model *struc.Model) (string, string, error) {
	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames
		fields     = model.FieldsTagValue
		varName    = goName(typeName+g.getIdentPart("TagValues"), *g.Conf.ExportVars)
	)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := baseType
	tagValueType := baseType
	tagValueArrayType := "[]" + tagValueType

	if *g.Conf.WrapType {
		tagValueType = g.getUsedTagValueType(typeName)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
		tagType = g.getUsedTagType(typeName)
	}

	varValue := "map[" + tagType + "]" + tagValueArrayType + "{\n"
	for _, tagName := range tagNames {
		constName := g.getUsedTagConstName(typeName, tagName)
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

		tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal)
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

func (g *Generator) generateTagFieldsMapVar(model *struc.Model) (string, string, error) {
	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames
		fields     = model.FieldsTagValue
		varName    = goName(typeName+g.getIdentPart("TagFields"), *g.Conf.ExportVars)
	)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := baseType
	fieldArrayType := "[]" + baseType

	if *g.Conf.WrapType {
		tagType = g.getUsedTagType(typeName)
		fieldArrayType = g.getFieldArrayType(typeName)
	}

	varValue := "map[" + tagType + "]" + fieldArrayType + "{\n"

	for _, tagName := range tagNames {
		constName := g.getUsedTagConstName(typeName, tagName)

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

			tagConstName := g.getUsedFieldConstName(typeName, fieldName)
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

func (g *Generator) generateTagFieldConstants(model *struc.Model, tagValueType string) error {

	if len(model.TagNames) == 0 {
		return g.noTagsError("Tag Fields Constants")
	}
	g.addConstDelim()
	for _, tagName := range model.TagNames {
		for _, fieldName := range model.FieldNames {
			if tagValue, ok := model.FieldsTagValue[fieldName][tagName]; ok {
				isEmptyTag := isEmpty(tagValue)
				if isEmptyTag {
					tagValue = fieldName
				}

				tagValueConstName := g.getTagValueConstName(model.TypeName, tagName, fieldName)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}

				constVal := g.getConstValue(tagValueType, tagValue)
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

func (g *Generator) generateFieldConstants(model *struc.Model, fieldType string, fieldNames []struc.FieldName) error {
	typeName := model.TypeName
	g.addConstDelim()
	for _, fieldName := range fieldNames {
		constName := g.getFieldConstName(typeName, fieldName, *g.Conf.Export)
		constVal := g.getConstValue(fieldType, fieldName)
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
		constName := g.getTagConstName(typeName, name)
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
	if _, constExists := g.constValues[constName]; !constExists {
		g.constNames = append(g.constNames, constName)
		g.constValues[constName] = constValue
	} else if existsValue, valueExists := g.constValues[constName]; valueExists {
		if existsValue != constValue {
			return errors.Errorf("duplicated constant with different values; const %v, values: %v, %v", constName, existsValue, constValue)
		}
	}
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

func (g *Generator) generateFieldsVar(model *struc.Model, fieldNames []struc.FieldName) (string, string, error) {
	typeName := model.TypeName
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

		constName := g.getUsedFieldConstName(typeName, fieldName)
		arrayVar += constName
		if !compact {
			arrayVar += ",\n"
		}
		i++
	}
	arrayVar += "}"

	varNameTemplate := typeName + g.getIdentPart("Fields")
	varName := goName(varNameTemplate, *g.Conf.ExportVars)
	return varName, arrayVar, nil
}

func (g *Generator) getFieldArrayType(typeName string) string {
	g.used.fieldArrayType = true
	return arrayType(g.getUsedFieldType(typeName))
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName) bool {
	_, excluded := g.excludedFields[fieldName]
	return (!*g.Conf.AllFields && !token.IsExported(string(fieldName))) || excluded
}

func (g *Generator) generateTagsVar(typeName string, tagNames []struc.TagName) (string, string, error) {
	varName := goName(typeName+g.getIdentPart("Tags"), *g.Conf.ExportVars)
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
		constName := g.getUsedTagConstName(typeName, tagName)
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
	return arrayType(g.getUsedTagType(typeName))
}

func (g *Generator) generateGetFieldValueFunc(model *struc.Model, packageName string) (string, string, string, error) {
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		fieldType  string
	)
	if *g.Conf.WrapType {
		fieldType = g.getUsedFieldType(typeName)
	} else {
		fieldType = baseType
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	funcName := g.renameFuncByConfig(goName("GetFieldValue", *g.Conf.Export))

	typeLink := g.typeName(typeName, packageName)

	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + fieldType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + fieldType + ") interface{}"
	}
	funcBody += " {" + g.noLint() + "\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}

		fieldExpr := g.transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName))
		funcBody += "case " + g.getUsedFieldConstName(typeName, fieldName) + ":\n" +
			"return " + fieldExpr + "\n"
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) transform(fieldName struc.FieldName, fieldType struc.FieldType, fieldRef string) string {
	var transforms []func(string) string
	if t, ok := g.transformValuesByFieldName[fieldName]; ok {
		transforms = append(transforms, t...)
	} else if t, ok = g.transformValuesByFieldType[fieldType]; ok {
		transforms = append(transforms, t...)
	} else {
		transforms = g.transformValues[:]
	}

	if len(transforms) == 0 {
		return fieldRef
	}
	for _, t := range transforms {
		before := fieldRef
		fieldRef = t(fieldRef)
		logger.Debugw("transforming field value: field %v, value before %v, after", fieldName, before, fieldRef)
	}
	return fieldRef
}

func (g *Generator) generateGetFieldValueByTagValueFunc(model *struc.Model, pkgAlias string) (string, string, string, error) {
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
	)

	funcName := g.renameFuncByConfig(goName("GetFieldValueByTagValue", *g.Conf.Export))
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}
	var valType string
	if *g.Conf.WrapType {
		valType = g.getUsedTagValueType(typeName)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	typeLink := g.typeName(typeName, pkgAlias)

	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + valType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + valType + ") interface{}"
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
				tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal)
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
			fieldType := model.FieldsType[fieldName]
			funcBody += "case " + caseExpr + ":\n" +
				"return " + g.transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName)) + "\n"
		}
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFuncGeneric(model *struc.Model, alias string) (string, string, string, error) {
	var (
		typeName = model.TypeName
		tagNames = model.TagNames
	)
	funcName := g.renameFuncByConfig(goName("GetFieldValuesByTag", *g.Conf.Export))
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	var tagType = baseType
	if *g.Conf.WrapType {
		tagType = g.getUsedTagType(typeName)
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	typeLink := g.typeName(typeName, alias)

	resultType := "[]interface{}"
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + tagType + ") " + resultType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + tagType + ") " + resultType
	}
	funcBody += " {" + g.noLint() + "\n" + "switch " + valVar + " {\n"

	for _, tagName := range tagNames {
		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, model)

		caseExpr := g.getUsedTagConstName(typeName, tagName)
		funcBody += "case " + caseExpr + ":\n" +
			"return " + fieldExpr + "\n"

	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFunctions(model *struc.Model, alias string) (string, []string, map[string]string, error) {

	getFuncName := func(funcNamePrefix string, tagName struc.TagName) string {
		return goName(funcNamePrefix+camel(string(tagName)), *g.Conf.Export)
	}

	var (
		typeName = model.TypeName
		tagNames = model.TagNames
		usedTags = g.getUsedTags(tagNames)
	)

	const funcNamePrefix = "GetFieldValuesByTag"
	if len(tagNames) == 0 {
		msg := ""
		for _, tagName := range usedTags {
			if len(msg) > 0 {
				msg += ","
			}
			msg += getFuncName(funcNamePrefix, tagName)
		}
		return "", nil, nil, g.noTagsError(msg)
	}

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	resultType := "[]interface{}"

	typeLink := g.typeName(typeName, alias)
	funcNames := make([]string, len(usedTags))
	funcBodies := make(map[string]string, len(usedTags))
	for i, tagName := range usedTags {
		funcName := g.renameFuncByConfig(getFuncName(funcNamePrefix, tagName))
		var funcBody string
		if *g.Conf.NoReceiver {
			funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") " + resultType
		} else {
			funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() " + resultType
		}
		funcBody += " {" + g.noLint() + "\n"

		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, model)

		funcBody += "return " + fieldExpr + "\n"
		funcBody += "}\n"

		funcNames[i] = funcName
		if _, ok := funcBodies[funcName]; ok {
			return "", nil, nil, errors.Errorf("duplicated function %s", funcName)
		}
		funcBodies[funcName] = funcBody
	}
	return typeLink, funcNames, funcBodies, nil
}

func (g *Generator) renameFuncByConfig(funcName string) string {
	if g.Conf.Name != nil && len(*g.Conf.Name) > 0 {
		renameTo := *g.Conf.Name
		logger.Debugw("rename func %v to %v", funcName, renameTo)
		funcName = renameTo
	}
	return funcName
}

func (g *Generator) fieldValuesArrayByTag(receiverRef string, resultType string, tagName struc.TagName, model *struc.Model) string {
	var (
		fieldNames     = model.FieldNames
		tagFieldValues = model.FieldsTagValue
	)
	fieldExpr := ""

	usedFieldNames := make([]struc.FieldName, 0)
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		_, ok := tagFieldValues[fieldName][tagName]
		if ok {
			usedFieldNames = append(usedFieldNames, fieldName)
		}
	}

	compact := *g.Conf.Compact || g.generatedAmount(usedFieldNames) <= oneLineSize
	if !compact {
		fieldExpr += "\n"
	}

	for _, fieldName := range usedFieldNames {
		if compact && len(fieldExpr) > 0 {
			fieldExpr += ", "
		}
		fieldType := model.FieldsType[fieldName]
		fieldExpr += g.transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName))
		if !compact {
			fieldExpr += ",\n"
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

func (g *Generator) generateAsMapFunc(model *struc.Model, pkg string) (string, string, string, error) {
	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	keyType := baseType
	if *g.Conf.WrapType {
		keyType = g.getUsedFieldType(model.TypeName)
	}

	funcName := g.renameFuncByConfig(goName("AsMap", *g.Conf.Export))
	typeLink := g.typeName(model.TypeName, pkg)
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") map[" + keyType + "]interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() map[" + keyType + "]interface{}"
	}
	funcBody += " {" + g.noLint() + "\n" +
		"	return map[" + keyType + "]interface{}{\n"

	for _, fieldName := range model.FieldNames {
		if g.isFieldExcluded(fieldName) {
			continue
		}
		funcBody += g.getUsedFieldConstName(model.TypeName, fieldName) + ": " +
			g.transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName)) + ",\n"
	}
	funcBody += "" +
		"	}\n" +
		"}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) generateAsTagMapFunc(model *struc.Model, alias string) (string, string, string, error) {
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
	)
	funcName := g.renameFuncByConfig(goName("AsTagMap", *g.Conf.Export))
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	receiverVar := "v"
	receiverRef := g.asRefIfNeed(receiverVar)

	tagValueType := baseType
	tagType := baseType
	if *g.Conf.WrapType {
		tagValueType = g.getUsedTagValueType(typeName)
		tagType = g.getUsedTagType(typeName)
	}

	valueType := "interface{}"

	varName := "tag"

	mapType := "map[" + tagValueType + "]" + valueType

	typeLink := g.typeName(typeName, alias)
	var funcBody string
	if *g.Conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + varName + " " + tagType + ") " + mapType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + varName + " " + tagType + ") " + mapType
	}

	funcBody += " {" + g.noLint() + "\n" +
		"switch " + varName + " {\n" +
		""

	for _, tagName := range tagNames {
		funcBody += "case " + g.getUsedTagConstName(typeName, tagName) + ":\n" +
			"return " + mapType + "{\n"
		for _, fieldName := range fieldNames {
			if g.isFieldExcluded(fieldName) {
				continue
			}
			tagVal, ok := fields[fieldName][tagName]

			if ok {
				tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}

				//if nestedModel := g.getNestedModel(model, fieldName); nestedModel != nil {
				//	for _,nestedFieldName:= range nestedModel.FieldNames {
				//		nestedFieldType := nestedModel.FieldsType[nestedFieldName]
				//		nestedTagValueConstName := tagValueConstName
				//
				//		g.getUsedFieldConstName(typeName, fieldPath)
				//
				//		funcBody += tagValueConstName + ": " + g.transform(fieldName, nestedFieldType, GetFieldRef(receiverRef, fieldName)) + ",\n"
				//	}
				//} else {
				fieldType := model.FieldsType[fieldName]
				funcBody += tagValueConstName + ": " + g.transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName)) + ",\n"
				//}
			}
		}

		funcBody += "}\n"
	}
	funcBody += "" +
		"	}\n" +
		"return nil" +
		"}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) typeName(typeName string, pkg string) string {
	if len(pkg) > 0 {
		return pkg + "." + typeName
	}
	return typeName
}

func (g *Generator) noTagsError(funcName string) error {
	includedTags := g.IncludedTags
	if len(includedTags) > 0 {
		return errors.Errorf(funcName+"; no tags for generating; included: %v", includedTags)
	} else {
		return errors.Errorf(funcName + "; no tags for generating;")
	}
}

func (g *Generator) getUsedTagConstName(typeName string, tag struc.TagName) string {
	if *g.Conf.HardcodeValues {
		return quoted(tag)
	}
	g.used.tagConstName = true
	return g.getTagConstName(typeName, tag)
}

func (g *Generator) getTagConstName(typeName string, tag struc.TagName) string {
	return goName(g.getTagType(typeName)+g.getIdentPart(tag), *g.Conf.Export)
}

func (g *Generator) getUsedTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, tagVal struc.TagValue) string {
	if *g.Conf.HardcodeValues {
		return quoted(tagVal)
	}
	g.used.tagValueConstName = true
	return g.getTagValueConstName(typeName, tag, fieldName)
}

func (g *Generator) getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	export := isExport(fieldName) && *g.Conf.Export
	return goName(g.getTagValueType(typeName)+g.getIdentPart(tag)+g.getIdentPart(fieldName), export)
}

func (g *Generator) getUsedFieldConstName(typeName string, fieldName struc.FieldName) string {
	if *g.Conf.HardcodeValues {
		return quoted(fieldName)
	}
	g.used.fieldConstName = true
	return g.getFieldConstName(typeName, fieldName, isExport(fieldName) && *g.Conf.Export)
}

func convertFieldPathToGoIdent(fieldName struc.FieldName) string {
	return strings.ReplaceAll(fieldName, ".", "")
}

func (g *Generator) generateConstants(str *struc.Model) error {
	data, err := g.NewTemplateDataObject(str)
	if err != nil {
		return err
	}

	for _, constName := range str.Constants {
		text, ok := str.ConstantTemplates[constName]
		if !ok {
			continue
		}
		constName = goName(constName, *g.Conf.Export)
		var constVal string
		if constVal, err = g.generateConst(constName, text, data); err != nil {
			return err
		} else if err = g.addConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) NewTemplateDataObject(str *struc.Model) (*TemplateDataObject, error) {
	if len(str.Constants) == 0 {
		return nil, nil
	}
	var (
		fieldsAmount = len(str.FieldNames)
		fields       = make([]string, fieldsAmount)
		tags         = make([]string, len(str.TagNames))
		fieldTypes   = make(map[string]string, fieldsAmount)
		fieldTags    = make(map[string][]string)
		tagFields    = make(map[string][]string)
		tagValues    = make(map[string][]string)
		ftv          = make(map[string]map[string]string)
	)

	for i, tagName := range str.TagNames {
		s := tagName
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
		fld := fieldName
		fields[i] = fld
		fieldTypes[fieldName] = str.FieldsType[fieldName]
		if g.isFieldExcluded(fieldName) {
			continue
		}
		t := make([]string, 0)
		for _, tagName := range str.TagNames {
			if v, ok := str.FieldsTagValue[fieldName][tagName]; ok {
				sv := v
				if g.excludedTagValues[sv] {
					continue
				}
				tg := tagName
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

	return &TemplateDataObject{
		Fields:        fields,
		Tags:          tags,
		FieldTypes:    fieldTypes,
		FieldTags:     fieldTags,
		TagValues:     tagValues,
		TagFields:     tagFields,
		FieldTagValue: ftv,
	}, nil
}

func (g *Generator) generateConst(constName string, constTemplate string, data *TemplateDataObject) (string, error) {
	add := func(first int, second int) int {
		return first + second
	}
	inc := func(value int) int {
		return add(value, 1)
	}
	dec := func(value int) int {
		return add(value, -1)
	}

	newMap := func(keyValues ...interface{}) (map[interface{}]interface{}, error) {
		if len(keyValues)%2 > 0 {
			return nil, errors.New("newMap has odd args amount")
		}
		m := map[interface{}]interface{}{}
		for i := 0; i < len(keyValues); i = i + 2 {
			m[keyValues[i]] = keyValues[i+1]
		}
		return m, nil
	}

	tmpl, err := template.New(constName).Funcs(template.FuncMap{"add": add, "inc": inc, "dec": dec, "hasValue": hasValue, "newMap": newMap}).Parse(constTemplate)
	if err != nil {
		return "", errors.Wrapf(err, "const: %s", constName)
	}

	buf := bytes.Buffer{}
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%v; context %v", err, data)
	}

	s := buf.String()
	if len(s) > 0 && s[0] == '`' {
		replaces := []map[string]string{{"\n": ""}, {"\t": ""}, {"\\t": "\t"}, {"\\n": "\n"}}
		for _, replace := range replaces {
			for replaceable, replacer := range replace {
				s = strings.ReplaceAll(s, replaceable, replacer)
			}
		}
	} else if s, err = g.splitLines(s); err != nil {
		return "", err
	}
	return s, nil
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
	for i, name := range names {
		if len(name) == 0 {
			if prev != nil && len(*prev) > 0 {
				newTypeNames = append(newTypeNames, name)
			}
		} else if _, ok := values[name]; ok {
			newTypeNames = append(newTypeNames, name)
			prev = &names[i]
		}
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

func (g *Generator) getFieldConstName(typeName string, fieldName struc.FieldName, export bool) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	return goName(g.getFieldType(typeName)+g.getIdentPart(fieldName), isExport(fieldName) && export)
}

func isExport(fieldName struc.FieldName) bool {
	return token.IsExported(fieldName)
}
