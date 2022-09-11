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
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"

	"golang.org/x/tools/go/packages"
)

const oneLineSize = 3

type Generator struct {
	name string

	outFile     *ast.File
	outFileInfo *token.File
	outPkg      *packages.Package
	// IncludedTags []struc.TagName
	// FoundTags    []struc.TagName

	outBuildTags string
	// outPackage   string
	// Conf         *Config
	// Content      *ContentConfig

	body *bytes.Buffer
	used Used

	excludedTagValues map[string]bool
	excludedFields    map[struc.FieldName]interface{}

	rewrite CodeRewriter

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

	imports map[string]string

	isRewrite bool
}

func New(name, outPackage, outBuildTags string, outFile *ast.File, outFileInfo *token.File, outPkg *packages.Package) *Generator {
	g := &Generator{
		// IncludedTags: includedTags,
		name: name,
		// outPackage:   outPackage,
		outBuildTags: outBuildTags,
		// Conf:         config.Generator,
		// Content:      config.Content,

		outFile:     outFile,
		outFileInfo: outFileInfo,
		outPkg:      outPkg,

		constNames:         make([]string, 0),
		constValues:        make(map[string]string),
		constComments:      make(map[string]string),
		varNames:           make([]string, 0),
		varValues:          make(map[string]string),
		typeNames:          make([]string, 0),
		typeValues:         make(map[string]string),
		funcNames:          make([]string, 0),
		funcValues:         make(map[string]string),
		receiverNames:      make([]string, 0),
		receiverFuncs:      make(map[string][]string),
		receiverFuncValues: make(map[string]map[string]string),

		imports: map[string]string{},

		excludedTagValues: make(map[string]bool),
		excludedFields:    make(map[struc.FieldName]interface{}),
	}
	g.isRewrite = g.IsRewrite(outFile, outFileInfo)
	return g
}

const DefaultConstLength = 80

type Config struct {
	Nolint         *bool
	Export         *bool
	NoReceiver     *bool
	ExportVars     *bool
	AllFields      *bool
	ReturnRefs     *bool
	WrapType       *bool
	HardcodeValues *bool
	NoEmptyTag     *bool
	Compact        *bool
	Snake          *bool
	Flat           *[]string
	ConstLength    *int
	ConstReplace   *[]string
	// IncludeFieldTags    *string
	Name                *string
	ExcludeFields       *[]string
	FieldValueRewriters *[]string
}

// func (c *Config) IncludedTags() (map[struc.TagName]struct{}, []struc.TagName) {
// 	var (
// 		includedTagArg  = *c.IncludeFieldTags
// 		includedTagsSet = make(map[struc.TagName]struct{})
// 		includedTags    = make([]struc.TagName, 0)
// 	)
// 	if len(includedTagArg) > 0 {
// 		includedTagNames := strings.Split(includedTagArg, ",")
// 		for _, includedTag := range includedTagNames {
// 			name := struc.TagName(includedTag)
// 			includedTagsSet[name] = struct{}{}
// 			includedTags = append(includedTags, name)
// 		}
// 	}
// 	return includedTagsSet, includedTags
// }

type ContentConfig struct {
	Constants       *[]string
	EnumFieldConsts *[]string

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

	// if len(*c.IncludeFieldTags) == 0 && len(*src.IncludeFieldTags) != 0 {
	// 	c.IncludeFieldTags = src.IncludeFieldTags
	// }

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

	// if len(*src.OutBuildTags) > 0 && len(*c.OutBuildTags) == 0 {
	// 	c.OutBuildTags = src.OutBuildTags
	// }

	if len(*src.Name) > 0 && len(*c.Name) == 0 {
		c.Name = src.Name
	}

	if len(*c.ExcludeFields) == 0 && len(*src.ExcludeFields) != 0 {
		c.ExcludeFields = src.ExcludeFields
	}

	if len(*c.FieldValueRewriters) == 0 && len(*src.FieldValueRewriters) != 0 {
		c.FieldValueRewriters = src.FieldValueRewriters
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

const BaseConstType = "string"

func OutPackageName(outPackageName string, outPackage *packages.Package) string {
	if len(outPackageName) == 0 {
		name := outPackage.Name
		if len(name) > 0 {
			outPackageName = name
		} else {
			outPackageName = packagePathToName(outPackage.PkgPath)
		}
		logger.Debugf("output package %v, path %v", outPackageName, outPackage.PkgPath)
	}
	return outPackageName
}

func (g *Generator) StructPackage(model *struc.Model) (string, error) {
	var (
		outFile    = g.outFile
		outPackage = g.outPkg
		needImport = model.PackagePath != outPackage.PkgPath
	)
	structPackage := ""
	if needImport {
		structPackage = model.PackageName
	}

	if !g.isRewrite && needImport {
		alias, found, err := g.findImportPackageAlias(model, outFile)
		if err != nil {
			return "", err
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
				if duplicated, err = HasDuplicatedPackage(outFile, structPackageSuffixed); err != nil {
					return "", err
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

	if needImport {
		importAlias := structPackage
		if name := packagePathToName(model.PackagePath); name == structPackage {
			importAlias = ""
		}
		g.AddImport(model.PackagePath, importAlias)
	}
	return structPackage, nil
}

func (g *Generator) GenerateFile(
	model *struc.Model, outFile *ast.File, outFileInfo *token.File, outPackage *packages.Package, structPackage, outPackageName string, isRewrite bool,
	conf Config, content ContentConfig,
) error {

	if *conf.NoEmptyTag {
		for fieldName, _tagNames := range model.FieldsTagValue {
			for tagName, tagValue := range _tagNames {
				tagValueConstName := g.getUsedTagValueConstName(model.TypeName, tagName, fieldName, tagValue, conf)
				if isEmpty(tagValue) {
					g.excludedTagValues[tagValueConstName] = true
				}
			}
		}
	}

	excludedFields := make(map[struc.FieldName]interface{})
	for _, excludes := range *conf.ExcludeFields {
		e := strings.Split(excludes, struc.ListValuesSeparator)
		for _, exclude := range e {
			excludedFields[exclude] = nil
		}
	}

	for _, fieldName := range model.FieldNames {
		if _, excluded := excludedFields[fieldName]; excluded {
			g.excludedFields[fieldName] = nil
		}
	}

	if err := g.generateConstants(model, *conf.ConstLength, *conf.Export, *conf.AllFields, *conf.Nolint); err != nil {
		return err
	}

	all := content.IsAll()

	if all || *content.Fields {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldsVar(model, model.FieldNames, conf)); err != nil {
			return err
		}
	}

	if all || *content.Tags {
		g.addVarDelim()
		if err := g.addVar(g.generateTagsVar(model.TypeName, model.TagNames, conf)); err != nil {
			return err
		}
	}

	if all || *content.FieldTagsMap {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldTagsMapVar(model.TypeName, model.TagNames, model.FieldNames, model.FieldsTagValue, conf)); err != nil {
			return err
		}
	}

	if all || len(*content.TagValues) > 0 {
		if len(model.TagNames) == 0 {
			return g.noTagsError("TagValues")
		}

		g.addVarDelim()
		values := *content.TagValues
		if len(*content.TagValues) == 0 {
			values = getTagsValues(model.TagNames)
		}
		vars, bodies, err := g.generateTagValuesVar(model.TypeName, values, model.FieldNames, model.FieldsTagValue, conf)
		if err != nil {
			return err
		}
		for _, varName := range vars {
			if err = g.addVar(varName, bodies[varName], nil); err != nil {
				return err
			}
		}
	}

	if all || *content.TagValuesMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagValuesMapVar(model, conf)); err != nil {
			return err
		}
	}

	if all || *content.TagFieldsMap {
		g.addVarDelim()
		if err := g.addVar(g.generateTagFieldsMapVar(model, conf)); err != nil {
			return err
		}
	}

	if all || *content.FieldTagValueMap {
		g.addVarDelim()
		if err := g.addVar(g.generateFieldTagValueMapVar(model, conf)); err != nil {
			return err
		}
	}

	getFieldValue := *content.GetFieldValue
	getFieldValueByTagValue := *content.GetFieldValueByTagValue
	getFieldValuesByTagGeneric := *content.GetFieldValuesByTagGeneric
	getFieldValuesByTag := *content.GetFieldValuesByTag
	asMap := *content.AsMap
	asTagMap := *content.AsTagMap

	generateManyFuncs := all || (toInt(getFieldValue)+toInt(getFieldValueByTagValue)+
		toInt(getFieldValuesByTagGeneric)+len(getFieldValuesByTag)+toInt(asMap)+toInt(asTagMap)) > 1
	if len(*conf.Name) > 0 && generateManyFuncs {
		return errors.New("-name not supported for multiple functions, please specify only one function")
	}

	if all || getFieldValue {
		if err := g.AddReceiverFunc(g.generateGetFieldValueFunc(model, structPackage, conf)); err != nil {
			return err
		}
	}
	if all || getFieldValueByTagValue {
		if err := g.AddReceiverFunc(g.generateGetFieldValueByTagValueFunc(model, structPackage, conf)); err != nil {
			return err
		}
	}

	if all || getFieldValuesByTagGeneric {
		if err := g.AddReceiverFunc(g.generateGetFieldValuesByTagFuncGeneric(model, structPackage, conf)); err != nil {
			return err
		}
	}

	if all || len(getFieldValuesByTag) > 0 {
		receiverType, funcNames, funcBodies, err := g.generateGetFieldValuesByTagFunctions(model, structPackage, conf, getFieldValuesByTag)
		if err != nil {
			return err
		}
		for _, funcName := range funcNames {
			funcBody := funcBodies[funcName]
			if err = g.AddReceiverFunc(receiverType, funcName, funcBody, nil); err != nil {
				return err
			}
		}
	}

	if all || asTagMap {
		if err := g.AddReceiverFunc(g.generateAsTagMapFunc(model, structPackage, conf)); err != nil {
			return err
		}
	}

	if err := g.generateHead(model, all, conf, content); err != nil {
		return err
	}

	noReceiver := conf.NoReceiver != nil && *conf.NoReceiver
	return g.WriteBody(outPackageName, noReceiver)
}

func (g *Generator) WriteBody(outPackageName string, noReceiver bool) error {
	if g.isRewrite {
		g.body = &bytes.Buffer{}
		g.writeHead(outPackageName)
		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeReceiverFunctions()
		g.writeFunctions()
	} else {
		//injects
		chunks, err := g.getInjectChunks(g.outFile, g.outFileInfo.Base(), noReceiver)
		if err != nil {
			return err
		}
		name := g.outFileInfo.Name()
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

func (g *Generator) IsRewrite(outFile *ast.File, outFileInfo *token.File) bool {
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

func HasDuplicatedPackage(outFile *ast.File, packageName string) (bool, error) {
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

func (g *Generator) getInjectChunks(outFile *ast.File, base int, noReceiver bool) (map[int]map[int]string, error) {
	chunks := make(map[int]map[int]string)

	importInjected := false
	for _, decl := range outFile.Decls {
		switch dt := decl.(type) {
		case *ast.GenDecl:
			if dt.Tok == token.IMPORT {
				var start int
				var end int
				if len(dt.Specs) == 0 {
					start = int(dt.Pos()) - base
					end = int(dt.End()) - base
				} else {
					if dt.Rparen != token.NoPos {
						start = int(dt.Rparen) - base
						end = start
					}
				}
				if expr := g.getImportsExpr(); len(expr) > 0 {
					chunks[start] = map[int]string{end: expr}
				}
				importInjected = true
			} else {
				for _, spec := range dt.Specs {
					switch st := spec.(type) {
					case *ast.TypeSpec:
						switch st.Type.(type) {
						case *ast.Ident, *ast.ArrayType:
						default:
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
					if handled, err = g.addReceiveFuncOnRewrite(params.List, name, chunks, start, end); err != nil {
						return nil, err
					}
				}
				if !handled {
					if funcDecl, hasFuncDecl := g.funcValues[name]; hasFuncDecl {
						chunks[start] = map[int]string{end: funcDecl}
						delete(g.funcValues, name)
					}
				}
			}
		}
	}

	if !importInjected {
		if expr := g.getImportsExpr(); len(expr) > 0 {
			start := int(outFile.Name.End()) - base
			end := start
			chunks[start] = map[int]string{end: expr}
		}
	}
	return chunks, nil
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
	if receiverFuncs, hasFuncs := g.receiverFuncValues[receiverName]; hasFuncs {
		funcDecl, hasFuncDecl := receiverFuncs[name]
		if hasFuncDecl {
			chunks[start] = map[int]string{end: funcDecl}
			delete(receiverFuncs, name)
			return true, nil
		}
	}
	return false, nil
}

func (g *Generator) getUsedTags(allTags []struc.TagName, getFieldValuesByTag []string) []struc.TagName {
	var usedTags []struc.TagName
	if len(getFieldValuesByTag) > 0 {
		usedTagNames := toSet(getFieldValuesByTag)
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

func (g *Generator) writeHead(packageName string) {
	g.writeBody("// %s'; DO NOT EDIT.\n\n", g.generatedMarker())
	g.writeBody(g.outBuildTags)
	g.writeBody("package %s\n", packageName)
	g.writeBody(g.getImportsExpr())
}

func (g *Generator) getImportsExpr() string {
	if len(g.imports) > 0 {
		return "\nimport (" + g.importsExprList() + "\n)\n"
	}
	return ""
}

func (g *Generator) importsExprList() string {
	imps := ""
	for pack, alias := range g.imports {
		if len(alias) == 0 {
			imps += "\n" + "\"" + pack + "\""
		} else {
			imps += "\n" + alias + "\"" + pack + "\""
		}
	}
	return imps
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
	if len(g.varNames) > 0 {
		g.writeBody("var(\n")
	}
	for _, name := range g.varNames {
		if len(name) == 0 {
			g.writeBody("\n")
			continue
		}
		value := g.varValues[name]
		g.writeBody("%v=%v", name, value)
		g.writeBody("\n")
	}
	if len(g.varNames) > 0 {
		g.writeBody(")\n")
	}
}

func (g *Generator) writeFunctions() {
	names := g.funcNames
	values := g.funcValues

	for _, name := range names {
		if value, ok := values[name]; ok {
			g.writeBody(value)
		}
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

func (g *Generator) generateHead(model *struc.Model, all bool, conf Config, content ContentConfig) error {
	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames

		fieldType  = BaseConstType
		tagType    = BaseConstType
		tagValType = BaseConstType

		usedFieldType    = g.used.fieldType || *content.EnumFields
		usedTagType      = g.used.tagType || *content.EnumTags
		usedTagValueType = g.used.tagValueType || *content.EnumTagValues
	)

	if usedFieldType {
		fieldType = GetFieldType(typeName, *conf.Export, *conf.Snake)
	}
	if usedTagType {
		tagType = g.getTagType(typeName, *conf.Export, *conf.Snake)
	}
	if usedTagValueType {
		tagValType = g.getTagValueType(typeName, *conf.Export, *conf.Snake)
	}

	wrapType := *conf.WrapType
	if wrapType {
		if usedFieldType || *content.EnumFields {
			_ = g.AddType(fieldType, BaseConstType)
			if g.used.fieldArrayType {
				_ = g.AddType(ArrayType(fieldType), "[]"+fieldType)
			}
		}

		if usedTagType {
			_ = g.AddType(tagType, BaseConstType)
			if g.used.tagArrayType {
				_ = g.AddType(ArrayType(tagType), "[]"+tagType)
			}
		}

		if usedTagValueType {
			tagValueType := tagValType
			_ = g.AddType(tagValueType, BaseConstType)
			if g.used.tagValueArrayType {
				_ = g.AddType(ArrayType(tagValueType), "[]"+tagValueType)
			}
		}
	}

	fieldConstName := g.used.fieldConstName || *content.EnumFields || all
	tagConstName := g.used.tagConstName || *content.EnumTags || all
	tagValueConstName := g.used.tagValueConstName || *content.EnumTagValues || all

	if fieldConstName {
		if err := g.GenerateFieldConstants(model, fieldType, fieldNames, *conf.Export, *conf.Snake, *conf.WrapType); err != nil {
			return err
		}
	}

	if tagConstName {
		if err := g.generateTagConstants(typeName, tagType, tagNames, conf); err != nil {
			return err
		}
	}

	if tagValueConstName {
		if err := g.generateTagFieldConstants(model, tagValType, conf); err != nil {
			return err
		}
	}

	if wrapType {
		if all || *content.Strings {
			if g.used.fieldArrayType {
				if err := g.addReceiverFuncWithImports(g.generateArrayToStringsFunc(ArrayType(fieldType), BaseConstType, conf)); err != nil {
					return err
				}
			}

			if g.used.tagArrayType {
				if err := g.addReceiverFuncWithImports(g.generateArrayToStringsFunc(ArrayType(tagType), BaseConstType, conf)); err != nil {
					return err
				}
			}

			if g.used.tagValueArrayType {
				if err := g.addReceiverFuncWithImports(g.generateArrayToStringsFunc(ArrayType(tagValType), BaseConstType, conf)); err != nil {
					return err
				}
			}
		}

		if *content.Excludes {
			if g.used.fieldArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, fieldType, ArrayType(fieldType), conf)
				if err := g.AddReceiverFunc(fieldType, funcName, funcBody, nil); err != nil {
					return err
				}
			}

			if g.used.tagArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagType, ArrayType(tagType), conf)
				if err := g.AddReceiverFunc(tagType, funcName, funcBody, nil); err != nil {
					return err
				}
			}

			if g.used.tagValueArrayType {
				funcName, funcBody := g.generateArrayToExcludesFunc(true, tagValType, ArrayType(tagValType), conf)
				if err := g.AddReceiverFunc(tagValType, funcName, funcBody, nil); err != nil {
					return err
				}
			}
		}
	} else {
		if *content.Excludes {
			funcName, funcBody := g.generateArrayToExcludesFunc(false, BaseConstType, "[]"+BaseConstType, conf)
			if err := g.AddFunc(funcName, funcBody); err != nil {
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

func (g *Generator) AddType(typeName string, typeValue string) error {
	if exists, ok := g.typeValues[typeName]; !ok {
		g.typeNames = append(g.typeNames, typeName)
		g.typeValues[typeName] = typeValue
	} else if typeValue != exists {
		return fmt.Errorf("duplicated type with different base type: type %s, expected base %s, actual %s",
			typeName, typeValue, exists)
	}
	return nil
}

func (g *Generator) generatedMarker() string {
	return fmt.Sprintf("Code generated by '%s", g.name)
}

func getUsedFieldType(typeName string, export, snake bool) string {
	return GetFieldType(typeName, export, snake)
}

func (g *Generator) getUsedTagType(typeName string, export, snake bool) string {
	g.used.tagType = true
	return g.getTagType(typeName, export, snake)
}

func (g *Generator) getUsedTagValueType(typeName string, export, snake bool) string {
	g.used.tagValueType = true
	return g.getTagValueType(typeName, export, snake)
}

func ArrayType(baseType string) string {
	return baseType + "List"
}

func (g *Generator) getTagValueType(typeName string, export, snake bool) string {
	return goName(typeName+getIdentPart("TagValue", snake), export)
}

func (g *Generator) getTagType(typeName string, export, snake bool) string {
	return goName(typeName+getIdentPart("Tag", snake), export)
}

func GetFieldType(typeName string, export, snake bool) string {
	return goName(typeName+getIdentPart("Field", snake), export)
}

func getIdentPart(suffix string, snake bool) string {
	if snake {
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

func (g *Generator) generateFieldTagValueMapVar(model *struc.Model, conf Config) (string, string, error) {
	var (
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
		typeName   = model.TypeName
	)

	var (
		export         = *conf.Export
		snake          = *conf.Snake
		wrapType       = *conf.WrapType
		hardcodeValues = *conf.HardcodeValues
		exportVars     = *conf.ExportVars
	)

	varName := goName(typeName+getIdentPart("FieldTagValue", snake), exportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	var varValue string
	fieldType := BaseConstType
	tagType := BaseConstType
	tagValueType := BaseConstType
	if wrapType {
		tagType = g.getUsedTagType(typeName, export, snake)
		fieldType = getUsedFieldType(typeName, export, snake)
		g.used.fieldType = true
		tagValueType = g.getUsedTagValueType(typeName, export, snake)

	}
	varValue = "map[" + fieldType + "]map[" + tagType + "]" + tagValueType + "{\n"
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}
		fieldConstName := g.getUsedFieldConstName(typeName, fieldName, hardcodeValues, export, snake)

		varValue += fieldConstName + ": map[" + tagType + "]" + tagValueType + "{"

		compact := *conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
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

			tagConstName := g.getUsedTagConstName(typeName, tagName, conf)
			tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal, conf)
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

func (g *Generator) generateFieldTagsMapVar(
	typeName string, tagNames []struc.TagName, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, conf Config,
) (string, string, error) {

	var (
		export         = *conf.Export
		snake          = *conf.Snake
		hardcodeValues = *conf.HardcodeValues
		exportVars     = *conf.ExportVars
	)

	varName := goName(typeName+getIdentPart("FieldTags", snake), exportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	fieldType := BaseConstType
	tagArrayType := "[]" + BaseConstType

	if *conf.WrapType {
		tagArrayType = g.getTagArrayType(typeName, export, snake)
		fieldType = getUsedFieldType(typeName, export, snake)
		g.used.fieldType = true
	}

	varValue := "map[" + fieldType + "]" + tagArrayType + "{\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}

		fieldConstName := g.getUsedFieldConstName(typeName, fieldName, hardcodeValues, export, snake)

		if *conf.WrapType {
			varValue += fieldConstName + ": " + tagArrayType + "{"
		} else {
			varValue += fieldConstName + ": []" + BaseConstType + "{"
		}

		compact := *conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
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
			tagConstName := g.getUsedTagConstName(typeName, tagName, conf)
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

func (g *Generator) generateTagValuesVar(
	typeName string, tagNames []string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, conf Config,
) ([]string, map[string]string, error) {

	vars := make([]string, 0)
	varValues := make(map[string]string)
	tagValueType := BaseConstType
	tagValueArrayType := "[]" + tagValueType
	if *conf.WrapType {
		tagValueType = g.getUsedTagValueType(typeName, *conf.Export, *conf.Snake)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
	}

	for _, tagName := range tagNames {
		varName := goName(typeName+getIdentPart("TagValues", *conf.Snake)+getIdentPart(string(tagName), *conf.Snake), *conf.ExportVars)
		valueBody := g.generateTagValueBody(typeName, tagValueArrayType, fieldNames, fields, struc.TagName(tagName), conf)
		vars = append(vars, varName)
		if _, ok := varValues[varName]; !ok {
			varValues[varName] = valueBody
		} else {
			return nil, nil, errors.Errorf("duplicated var %s", varName)
		}
	}

	return vars, varValues, nil
}

func (g *Generator) generateTagValuesMapVar(model *struc.Model, conf Config) (string, string, error) {
	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames
		fields     = model.FieldsTagValue
		varName    = goName(typeName+getIdentPart("TagValues", *conf.Snake), *conf.ExportVars)
	)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := BaseConstType
	tagValueType := BaseConstType
	tagValueArrayType := "[]" + tagValueType

	if *conf.WrapType {
		tagValueType = g.getUsedTagValueType(typeName, *conf.Export, *conf.Snake)
		tagValueArrayType = g.getTagValueArrayType(tagValueType)
		tagType = g.getUsedTagType(typeName, *conf.Export, *conf.Snake)
	}

	varValue := "map[" + tagType + "]" + tagValueArrayType + "{\n"
	for _, tagName := range tagNames {
		constName := g.getUsedTagConstName(typeName, tagName, conf)
		valueBody := g.generateTagValueBody(typeName, tagValueArrayType, fieldNames, fields, tagName, conf)
		varValue += constName + ": " + valueBody + ",\n"
	}
	varValue += "}"

	return varName, varValue, nil
}

func (g *Generator) generateTagValueBody(
	typeName string, tagValueArrayType string, fieldNames []struc.FieldName, fields map[struc.FieldName]map[struc.TagName]struc.TagValue, tagName struc.TagName,
	conf Config,
) string {
	var varValue string
	if *conf.WrapType {
		varValue += tagValueArrayType + "{"
	} else {
		varValue += "[]" + BaseConstType + "{"
	}

	compact := *conf.Compact || g.generatedAmount(fieldNames, conf) <= oneLineSize
	if !compact {
		varValue += "\n"
	}

	ti := 0
	for _, fieldName := range fieldNames {
		tagVal, ok := fields[fieldName][tagName]
		if !ok {
			continue
		}

		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}

		if compact && ti > 0 {
			varValue += ", "
		}

		tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal, conf)
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
	return ArrayType(tagValueType)
}

func (g *Generator) generateTagFieldsMapVar(model *struc.Model, conf Config) (string, string, error) {
	var (
		export         = *conf.Export
		snake          = *conf.Snake
		hardcodeValues = *conf.HardcodeValues
		exportVars     = *conf.ExportVars
	)

	var (
		typeName   = model.TypeName
		tagNames   = model.TagNames
		fieldNames = model.FieldNames
		fields     = model.FieldsTagValue
		varName    = goName(typeName+getIdentPart("TagFields", snake), exportVars)
	)

	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagType := BaseConstType
	fieldArrayType := "[]" + BaseConstType

	if *conf.WrapType {
		tagType = g.getUsedTagType(typeName, export, snake)
		fieldArrayType = g.getFieldArrayType(typeName, export, snake)
	}

	varValue := "map[" + tagType + "]" + fieldArrayType + "{\n"

	for _, tagName := range tagNames {
		constName := g.getUsedTagConstName(typeName, tagName, conf)

		varValue += constName + ": " + fieldArrayType + "{"

		compact := *conf.Compact || g.generatedAmount(fieldNames, conf) <= oneLineSize
		if !compact {
			varValue += "\n"
		}

		ti := 0
		for _, fieldName := range fieldNames {
			_, ok := fields[fieldName][tagName]
			if !ok {
				continue
			}
			if g.isFieldExcluded(fieldName, *conf.AllFields) {
				continue
			}

			if compact && ti > 0 {
				varValue += ", "
			}

			tagConstName := g.getUsedFieldConstName(typeName, fieldName, hardcodeValues, export, snake)
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

func (g *Generator) generateTagFieldConstants(model *struc.Model, tagValueType string, conf Config) error {
	if len(model.TagNames) == 0 {
		return g.noTagsError("Tag Fields Constants")
	}
	g.AddConstDelim()
	for _, tagName := range model.TagNames {
		for _, fieldName := range model.FieldNames {
			if tagValue, ok := model.FieldsTagValue[fieldName][tagName]; ok {
				isEmptyTag := isEmpty(tagValue)
				if isEmptyTag {
					tagValue = fieldName
				}

				tagValueConstName := g.getTagValueConstName(model.TypeName, tagName, fieldName, conf)
				if g.excludedTagValues[tagValueConstName] {
					continue
				}

				constVal := g.GetConstValue(tagValueType, tagValue, *conf.WrapType)
				if err := g.AddConst(tagValueConstName, constVal); err != nil {
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

func (g *Generator) generateTagConstants(typeName string, tagType string, tagNames []struc.TagName, conf Config) error {
	if len(tagNames) == 0 {
		return g.noTagsError("Tag Constants")
	}
	g.AddConstDelim()
	for _, name := range tagNames {
		constName := g.getTagConstName(typeName, name, conf)
		constVal := g.GetConstValue(tagType, string(name), *conf.WrapType)
		if err := g.AddConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) AddConstDelim() {
	if len(g.constNames) > 0 {
		g.constNames = append(g.constNames, "")
	}
}

func (g *Generator) AddFunсDelim() {
	if len(g.funcNames) > 0 {
		g.funcNames = append(g.funcNames, "")
	}
}

func (g *Generator) AddConst(constName, constValue string) error {
	if exists, ok := g.constValues[constName]; ok && exists != constValue {
		return errors.Errorf("duplicated constant with different value; const %v, values: %v, %v", constName, exists, constValue)
	} else {
		g.constNames = append(g.constNames, constName)
		g.constValues[constName] = constValue
	}
	return nil
}

func (g *Generator) GetConstValue(typ string, value string, wrapType bool) (constValue string) {
	if wrapType {
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

func (g *Generator) AddFunc(funcName, funcValue string) error {
	if exists, ok := g.funcValues[funcName]; ok && exists != funcValue {
		return errors.Errorf("duplicated func with different value; const %v, values: %v, %v", funcName, exists, funcValue)
	}
	g.funcNames = append(g.funcNames, funcName)
	g.funcValues[funcName] = funcValue
	return nil
}

func (g *Generator) addReceiverFuncWithImports(receiverName, funcName, funcValue string, imports map[string]string, err error) error {
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

	for pack, alias := range imports {
		g.AddImport(pack, alias)
	}
	return nil
}

func (g *Generator) AddImport(pack, alias string) {
	if exists, ok := g.imports[pack]; ok {
		logger.Debugf("replace imported package %s by %s, alias %s", exists, pack, alias)
	}
	g.imports[pack] = alias
}

func (g *Generator) AddReceiverFunc(receiverName, funcName, funcValue string, err error) error {
	return g.addReceiverFuncWithImports(receiverName, funcName, funcValue, nil, err)
}

func (g *Generator) generateFieldsVar(model *struc.Model, fieldNames []struc.FieldName, conf Config) (string, string, error) {
	var (
		export         = *conf.Export
		snake          = *conf.Snake
		wrapType       = *conf.WrapType
		allFields      = *conf.AllFields
		hardcodeValues = *conf.HardcodeValues
		compact        = *conf.Compact
		exportVars     = *conf.ExportVars
	)

	typeName := model.TypeName
	var arrayVar string
	if wrapType {
		arrayVar = g.getFieldArrayType(typeName, export, snake) + "{"
	} else {
		arrayVar = "[]" + BaseConstType + "{"
	}

	compact = compact || g.generatedAmount(fieldNames, conf) <= oneLineSize
	if !compact {
		arrayVar += "\n"
	}

	i := 0
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, allFields) {
			continue
		}

		if compact && i > 0 {
			arrayVar += ", "
		}

		constName := g.getUsedFieldConstName(typeName, fieldName, hardcodeValues, export, snake)
		arrayVar += constName
		if !compact {
			arrayVar += ",\n"
		}
		i++
	}
	arrayVar += "}"

	varNameTemplate := typeName + getIdentPart("Fields", snake)
	varName := goName(varNameTemplate, exportVars)
	return varName, arrayVar, nil
}

func (g *Generator) getFieldArrayType(typeName string, export, snake bool) string {
	g.used.fieldArrayType = true
	g.used.fieldType = true
	return ArrayType(getUsedFieldType(typeName, export, snake))
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName, allFields bool) bool {
	_, excluded := g.excludedFields[fieldName]
	return IsFieldExcluded(fieldName, allFields) || excluded
}

func IsFieldExcluded(fieldName struc.FieldName, includePrivate bool) bool {
	return (!includePrivate && !token.IsExported(string(fieldName)))
}

func (g *Generator) generateTagsVar(typeName string, tagNames []struc.TagName, conf Config) (string, string, error) {
	varName := goName(typeName+getIdentPart("Tags", *conf.Snake), *conf.ExportVars)
	if len(tagNames) == 0 {
		return "", "", g.noTagsError(varName)
	}

	tagArrayType := "[]" + BaseConstType

	if *conf.WrapType {
		tagArrayType = g.getTagArrayType(typeName, *conf.Export, *conf.Snake)
	}

	arrayVar := tagArrayType + "{"

	compact := *conf.Compact || len(tagNames) <= oneLineSize

	if !compact {
		arrayVar += "\n"
	}

	for i, tagName := range tagNames {
		if compact && i > 0 {
			arrayVar += ", "
		}
		constName := g.getUsedTagConstName(typeName, tagName, conf)
		arrayVar += constName

		if !compact {
			arrayVar += ",\n"
		}
	}
	arrayVar += "}"

	return varName, arrayVar, nil
}

func (g *Generator) getTagArrayType(typeName string, export, snake bool) string {
	g.used.tagArrayType = true
	return ArrayType(g.getUsedTagType(typeName, export, snake))
}

func (g *Generator) generateGetFieldValueFunc(model *struc.Model, packageName string, conf Config) (string, string, string, error) {
	var (
		export         = *conf.Export
		snake          = *conf.Snake
		wrapType       = *conf.WrapType
		allFields      = *conf.AllFields
		hardcodeValues = *conf.HardcodeValues
		noReceiver     = *conf.NoReceiver
		returnRefs     = *conf.ReturnRefs
		nolint         = *conf.Nolint
		name           = *conf.Name
	)
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		fieldType  string
	)
	if wrapType {
		g.used.fieldType = true
		fieldType = getUsedFieldType(typeName, export, snake)
	} else {
		fieldType = BaseConstType
	}

	valVar := "field"
	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, returnRefs)

	funcName := renameFuncByConfig(goName("GetFieldValue", export), name)

	typeLink := getTypeName(typeName, packageName)

	var funcBody string
	if noReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + fieldType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + fieldType + ") interface{}"
	}
	funcBody += " {" + g.noLint(nolint) + "\n" + "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, allFields) {
			continue
		}

		fieldExpr := g.Transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName))
		funcBody += "case " + g.getUsedFieldConstName(typeName, fieldName, hardcodeValues, export, snake) + ":\n" +
			"return " + fieldExpr + "\n"
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) Transform(fieldName struc.FieldName, fieldType struc.FieldType, fieldRef string) string {
	return g.rewrite.Transform(fieldName, fieldType, fieldRef)
}

func (g *Generator) generateGetFieldValueByTagValueFunc(model *struc.Model, pkgAlias string, conf Config) (string, string, string, error) {
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
	)

	funcName := renameFuncByConfig(goName("GetFieldValueByTagValue", *conf.Export), *conf.Name)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}
	var valType string
	if *conf.WrapType {
		valType = g.getUsedTagValueType(typeName, *conf.Export, *conf.Snake)
	} else {
		valType = "string"
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, *conf.ReturnRefs)

	typeLink := getTypeName(typeName, pkgAlias)

	var funcBody string
	if *conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + valType + ") interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + valType + ") interface{}"
	}
	funcBody += " {" + g.noLint(*conf.Nolint) + "\n"
	funcBody += "switch " + valVar + " {\n"

	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}

		var caseExpr string

		compact := *conf.Compact || g.generategAmount(tagNames, fields, fieldName) <= oneLineSize
		if !compact {
			caseExpr += "\n"
		}
		for _, tagName := range tagNames {
			tagVal, ok := fields[fieldName][tagName]
			if ok {
				tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal, conf)
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
				"return " + g.Transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName)) + "\n"
		}
	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFuncGeneric(model *struc.Model, alias string, conf Config) (string, string, string, error) {
	var (
		typeName = model.TypeName
		tagNames = model.TagNames
	)
	funcName := renameFuncByConfig(goName("GetFieldValuesByTag", *conf.Export), *conf.Name)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	var tagType = BaseConstType
	if *conf.WrapType {
		tagType = g.getUsedTagType(typeName, *conf.Export, *conf.Snake)
	}

	valVar := "tag"
	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, *conf.ReturnRefs)

	typeLink := getTypeName(typeName, alias)

	resultType := "[]interface{}"
	var funcBody string
	if *conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + valVar + " " + tagType + ") " + resultType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + valVar + " " + tagType + ") " + resultType
	}
	funcBody += " {" + g.noLint(*conf.Nolint) + "\n" + "switch " + valVar + " {\n"

	for _, tagName := range tagNames {
		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, model, conf)

		caseExpr := g.getUsedTagConstName(typeName, tagName, conf)
		funcBody += "case " + caseExpr + ":\n" +
			"return " + fieldExpr + "\n"

	}

	funcBody += "}\n" +
		"return nil" +
		"\n}\n"

	return typeLink, funcName, funcBody, nil
}

func (g *Generator) generateGetFieldValuesByTagFunctions(model *struc.Model, alias string, conf Config, getFieldValuesByTag []string) (string, []string, map[string]string, error) {

	getFuncName := func(funcNamePrefix string, tagName struc.TagName) string {
		return goName(funcNamePrefix+camel(string(tagName)), *conf.Export)
	}

	var (
		typeName = model.TypeName
		tagNames = model.TagNames
		usedTags = g.getUsedTags(tagNames, getFieldValuesByTag)
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
	receiverRef := AsRefIfNeed(receiverVar, *conf.ReturnRefs)

	resultType := "[]interface{}"

	typeLink := getTypeName(typeName, alias)
	funcNames := make([]string, len(usedTags))
	funcBodies := make(map[string]string, len(usedTags))
	for i, tagName := range usedTags {
		funcName := renameFuncByConfig(getFuncName(funcNamePrefix, tagName), *conf.Name)
		var funcBody string
		if *conf.NoReceiver {
			funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") " + resultType
		} else {
			funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() " + resultType
		}
		funcBody += " {" + g.noLint(*conf.Nolint) + "\n"

		fieldExpr := g.fieldValuesArrayByTag(receiverRef, resultType, tagName, model, conf)

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

func renameFuncByConfig(funcName, renameTo string) string {
	if len(renameTo) > 0 {
		logger.Debugw("rename func %v to %v", funcName, renameTo)
		funcName = renameTo
	}
	return funcName
}

func (g *Generator) fieldValuesArrayByTag(receiverRef string, resultType string, tagName struc.TagName, model *struc.Model, conf Config) string {
	var (
		fieldNames     = model.FieldNames
		tagFieldValues = model.FieldsTagValue
	)
	fieldExpr := ""

	usedFieldNames := make([]struc.FieldName, 0)
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}
		_, ok := tagFieldValues[fieldName][tagName]
		if ok {
			usedFieldNames = append(usedFieldNames, fieldName)
		}
	}

	compact := *conf.Compact || g.generatedAmount(usedFieldNames, conf) <= oneLineSize
	if !compact {
		fieldExpr += "\n"
	}

	for _, fieldName := range usedFieldNames {
		if compact && len(fieldExpr) > 0 {
			fieldExpr += ", "
		}
		fieldType := model.FieldsType[fieldName]
		fieldExpr += g.Transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName))
		if !compact {
			fieldExpr += ",\n"
		}
	}
	fieldExpr = resultType + "{" + fieldExpr + "}"
	return fieldExpr
}

func (g *Generator) generatedAmount(fieldNames []struc.FieldName, conf Config) int {
	l := 0
	for _, fieldName := range fieldNames {
		if g.isFieldExcluded(fieldName, *conf.AllFields) {
			continue
		}
		l++
	}
	return l
}

func AsRefIfNeed(receiverVar string, returnRefs bool) string {
	receiverRef := receiverVar
	if returnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func (g *Generator) generateArrayToExcludesFunc(receiver bool, typeName, arrayTypeName string, conf Config) (string, string) {
	funcName := goName("Excludes", *conf.Export)
	receiverVar := "v"
	funcDecl := "func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "(excludes ..." + typeName + ") " + arrayTypeName + " {" + g.noLint(*conf.Nolint) + "\n"
	if !receiver {
		receiverVar = "values"
		funcDecl = "func " + funcName + " (" + receiverVar + " " + arrayTypeName + ", excludes ..." + typeName + ") " + arrayTypeName + " {" + g.noLint(*conf.Nolint) + "\n"
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

func (g *Generator) generateArrayToStringsFunc(arrayTypeName string, resultType string, conf Config) (string, string, string, map[string]string, error) {
	funcName := goName("Strings", *conf.Export)
	receiverVar := "v"
	funcBody := "" +
		"func (" + receiverVar + " " + arrayTypeName + ") " + funcName + "() []" + resultType + " {" + g.noLint(*conf.Nolint) + "\n" +
		"		return *(*[]string)(unsafe.Pointer(&" + receiverVar + "))\n" +
		"	}\n"
	return arrayTypeName, funcName, funcBody, map[string]string{"unsafe": ""}, nil
}

func (g *Generator) generateAsTagMapFunc(model *struc.Model, alias string, conf Config) (string, string, string, error) {
	var (
		typeName   = model.TypeName
		fieldNames = model.FieldNames
		tagNames   = model.TagNames
		fields     = model.FieldsTagValue
	)
	funcName := renameFuncByConfig(goName("AsTagMap", *conf.Export), *conf.Name)
	if len(tagNames) == 0 {
		return "", "", "", g.noTagsError(funcName)
	}

	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, *conf.ReturnRefs)

	tagValueType := BaseConstType
	tagType := BaseConstType
	if *conf.WrapType {
		tagValueType = g.getUsedTagValueType(typeName, *conf.Export, *conf.Snake)
		tagType = g.getUsedTagType(typeName, *conf.Export, *conf.Snake)
	}

	valueType := "interface{}"

	varName := "tag"

	mapType := "map[" + tagValueType + "]" + valueType

	typeLink := getTypeName(typeName, alias)
	var funcBody string
	if *conf.NoReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ", " + varName + " " + tagType + ") " + mapType
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "(" + varName + " " + tagType + ") " + mapType
	}

	funcBody += " {" + g.noLint(*conf.Nolint) + "\n" +
		"switch " + varName + " {\n" +
		""

	for _, tagName := range tagNames {
		funcBody += "case " + g.getUsedTagConstName(typeName, tagName, conf) + ":\n" +
			"return " + mapType + "{\n"
		for _, fieldName := range fieldNames {
			if g.isFieldExcluded(fieldName, *conf.AllFields) {
				continue
			}
			tagVal, ok := fields[fieldName][tagName]

			if ok {
				tagValueConstName := g.getUsedTagValueConstName(typeName, tagName, fieldName, tagVal, conf)
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
				funcBody += tagValueConstName + ": " + g.Transform(fieldName, fieldType, struc.GetFieldRef(receiverRef, fieldName)) + ",\n"
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

func getTypeName(typeName string, pkg string) string {
	if len(pkg) > 0 {
		return pkg + "." + typeName
	}
	return typeName
}

func (g *Generator) noTagsError(funcName string) error {
	// includedTags := g.IncludedTags
	// if len(includedTags) > 0 {
	// return errors.Errorf(funcName+"; no tags for generating; included: %v", includedTags)
	// } else {
	return errors.Errorf(funcName + "; no tags for generating;")
	// }
}

func (g *Generator) getUsedTagConstName(typeName string, tag struc.TagName, conf Config) string {
	if *conf.HardcodeValues {
		return quoted(tag)
	}
	g.used.tagConstName = true
	return g.getTagConstName(typeName, tag, conf)
}

func (g *Generator) getTagConstName(typeName string, tag struc.TagName, conf Config) string {
	return goName(g.getTagType(typeName, *conf.Export, *conf.Snake)+getIdentPart(tag, *conf.Snake), *conf.Export)
}

func (g *Generator) getUsedTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, tagVal struc.TagValue, conf Config) string {
	if *conf.HardcodeValues {
		return quoted(tagVal)
	}
	g.used.tagValueConstName = true
	return g.getTagValueConstName(typeName, tag, fieldName, conf)
}

func (g *Generator) getTagValueConstName(typeName string, tag struc.TagName, fieldName struc.FieldName, conf Config) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	export := isExport(fieldName) && *conf.Export
	return goName(g.getTagValueType(typeName, *conf.Export, *conf.Snake)+getIdentPart(tag, *conf.Snake)+getIdentPart(fieldName, *conf.Snake), export)
}

func (g *Generator) GetTagTemplateConstName(typeName string, fieldName struc.FieldName, tags []struc.TagName, export, snake bool) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	export = isExport(fieldName) && export
	tagsPart := ""
	for _, tag := range tags {
		tagsPart += getIdentPart(tag, snake)
	}
	return goName(typeName+tagsPart+getIdentPart(fieldName, snake), export)
}

func (g *Generator) getUsedFieldConstName(typeName string, fieldName struc.FieldName, hardcodeValues, export, snake bool) string {
	if hardcodeValues {
		return quoted(fieldName)
	}
	g.used.fieldConstName = true
	return GetFieldConstName(typeName, fieldName, isExport(fieldName) && export, snake)
}

func convertFieldPathToGoIdent(fieldName struc.FieldName) string {
	return strings.ReplaceAll(fieldName, ".", "")
}

func (g *Generator) generateConstants(str *struc.Model, constLength int, export, allFields, nolint bool) error {
	data, err := g.NewTemplateDataObject(str, allFields)
	if err != nil {
		return err
	}

	for _, constName := range str.Constants {
		text, ok := str.ConstantTemplates[constName]
		if !ok {
			continue
		}
		constName = goName(constName, export)
		var constVal string
		if constVal, err = g.generateConst(constName, text, data, constLength, nolint); err != nil {
			return err
		} else if err = g.AddConst(constName, constVal); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) NewTemplateDataObject(str *struc.Model, allFields bool) (*TemplateDataObject, error) {
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
			if g.isFieldExcluded(fieldName, allFields) {
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
		if g.isFieldExcluded(fieldName, allFields) {
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

func (g *Generator) generateConst(constName string, constTemplate string, data *TemplateDataObject, constLength int, nolint bool) (string, error) {
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

	tmpl, err := template.New(constName).Funcs(template.FuncMap{"add": add, "inc": inc, "dec": dec, "contains": contains, "newMap": newMap}).Parse(constTemplate)
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
	} else if s, err = g.splitLines(s, constLength, nolint); err != nil {
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

func (g *Generator) noLint(nolint bool) string {
	if nolint {
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

func (g *Generator) splitLines(generatedValue string, stepSize int, nolint bool) (string, error) {
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
								buf.WriteString(g.noLint(nolint))
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
								buf.WriteString(g.noLint(nolint))
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

func GetFieldConstName(typeName string, fieldName struc.FieldName, export, snake bool) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	return goName(GetFieldType(typeName, export, snake)+getIdentPart(fieldName, snake), isExport(fieldName) && export)
}

func isExport(fieldName struc.FieldName) bool {
	return token.IsExported(fieldName)
}
