package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

const oneLineSize = 3

const Autoname = "."

type Generator struct {
	name string

	outFile      *ast.File
	outFileInfo  *token.File
	outPkg       *packages.Package
	outBuildTags string

	body *bytes.Buffer

	excludedTagValues map[string]bool

	rewrite CodeRewriter

	constNames []string
	constants  constants

	varNames   []string
	varValues  map[string]string
	typeNames  []string
	typeValues map[string]string
	funcNames  []string
	funcValues funcBodies

	imports map[string]string

	rewriteOutFile bool
}

type funcBodies map[string]funcBody

func (c funcBodies) names() map[string]string {
	r := make(map[string]string, len(c))
	for k, _ := range c {
		r[k] = k
	}
	return r
}

type funcBody struct {
	body string
	node *ast.FuncDecl
}

func (f *funcBody) String() (string, error) {
	if len(f.body) > 0 {
		return f.body, nil
	}
	return stringifyAst(f.node)
}

type constant struct {
	typ, value, comment string
}

type constants map[string]constant

func (c constants) nvMap() map[string]string {
	r := make(map[string]string, len(c))
	for k, v := range c {
		r[k] = v.value
	}
	return r
}

func New(name, outBuildTags string, outFile *ast.File, outFileInfo *token.File, outPkg *packages.Package) *Generator {
	g := &Generator{
		name:              name,
		outBuildTags:      outBuildTags,
		outFile:           outFile,
		outFileInfo:       outFileInfo,
		outPkg:            outPkg,
		constNames:        make([]string, 0),
		constants:         make(map[string]constant),
		varNames:          make([]string, 0),
		varValues:         make(map[string]string),
		typeNames:         make([]string, 0),
		typeValues:        make(map[string]string),
		funcNames:         make([]string, 0),
		funcValues:        make(map[string]funcBody),
		imports:           map[string]string{},
		excludedTagValues: make(map[string]bool),
	}
	g.rewriteOutFile = g.IsRewrite(outFile, outFileInfo)
	return g
}

const DefaultConstLength = 80

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

	if !g.rewriteOutFile && needImport {
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
				if duplicated, err = hasDuplicatedPackage(outFile, structPackageSuffixed); err != nil {
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

func (g *Generator) WriteBody(outPackageName string) error {
	if g.rewriteOutFile {
		g.body = &bytes.Buffer{}
		if err := g.writeHead(outPackageName); err != nil {
			return err
		}
		if err := g.writeTypes(); err != nil {
			return err
		}
		if err := g.writeConstants(); err != nil {
			return err
		}
		g.writeVars()
		if err := g.writeFunctions(); err != nil {
			return err
		}
	} else {
		//injects
		chunks, err := g.getInjectChunks(g.outFile, g.outFileInfo.Base())
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

		if err := g.writeTypes(); err != nil {
			return err
		}
		if err := g.writeConstants(); err != nil {
			return err
		}
		g.writeVars()
		if err := g.writeFunctions(); err != nil {
			return err
		}
	}
	return nil
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

func hasDuplicatedPackage(outFile *ast.File, packageName string) (bool, error) {
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

func (g *Generator) getInjectChunks(outFile *ast.File, base int) (map[int]map[int]string, error) {
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
				if expr, err := g.getImportsExpr(); err != nil {
					return nil, err
				} else if len(expr) > 0 {
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
							name := objectName.Name
							var generatingValues map[string]string
							switch dt.Tok {
							case token.TYPE:
								generatingValues = g.typeValues
							case token.CONST:
								if constant, found := g.constants[name]; found {
									chunks[start] = map[int]string{end: constant.value}
									delete(g.constants, name)
								}
							case token.VAR:
								generatingValues = g.varValues
							}
							if generatingValues != nil {
								if newValue, found := generatingValues[name]; found {
									chunks[start] = map[int]string{end: newValue}
									delete(generatingValues, name)
								}
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
			} else if f, ok := g.funcValues[name]; ok {
				if b, err := f.String(); err != nil {
					return nil, err
				} else {
					chunks[start] = map[int]string{end: b}
					delete(g.funcValues, name)
				}
			}
		}
	}

	if !importInjected {
		if expr, err := g.getImportsExpr(); err != nil {
			return nil, err
		} else if len(expr) > 0 {
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
	mName := MethodName(receiverName, name)
	if f, ok := g.funcValues[mName]; ok {
		if b, err := f.String(); err != nil {
			return false, err
		} else {
			chunks[start] = map[int]string{end: b}
			delete(g.funcValues, mName)
			return true, nil
		}

	}
	return false, nil
}

func MethodName(typ, fun string) string { return typ + "." + fun }

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

func (g *Generator) writeHead(packageName string) error {
	g.writeBody("// %s'; DO NOT EDIT.\n\n", g.generatedMarker())
	g.writeBody(g.outBuildTags)
	g.writeBody("package %s\n", packageName)
	if imps, err := g.getImportsExpr(); err != nil {
		return err
	} else {
		g.writeBody(imps)
		g.writeBody("\n")
	}
	return nil
}

func (g *Generator) getImportsExpr() (string, error) {
	fset := token.NewFileSet()
	out := &bytes.Buffer{}
	imports := g.getImports()
	if imports == nil {
		return "", nil
	}
	if err := printer.Fprint(out, fset, imports); err != nil {
		return "", err
	}
	return out.String(), nil

}

func (g *Generator) getImports() *ast.GenDecl {
	specs := []ast.Spec{}
	for pack, alias := range g.imports {
		specs = append(specs, newImport(alias, pack))
	}
	if len(specs) == 0 {
		return nil
	}
	return &ast.GenDecl{Tok: token.IMPORT, Specs: specs}
}

func (g *Generator) writeSpecs(specs ast.Node) error {
	if specs == nil || (reflect.ValueOf(specs).Kind() == reflect.Ptr && reflect.ValueOf(specs).IsNil()) {
		return nil
	}
	fset := token.NewFileSet()
	if err := printer.Fprint(g.body, fset, specs); err != nil {
		return err
	}
	g.writeBody("\n")
	return nil
}

func (g *Generator) writeConstants() error {
	return g.writeSpecs(g.getConstants())
}

func (g *Generator) getConstants() *ast.GenDecl {
	specs := []ast.Spec{}
	typ := BaseConstType
	for _, name := range g.constNames {
		if len(name) == 0 {
			continue
		}
		constant := g.constants[name]
		typ = constant.typ
		writeType := typ != BaseConstType
		var c *ast.ValueSpec
		if writeType {
			c = newConst(name, constant.typ, constant.value, constant.comment)
		} else {
			c = newConst(name, "", constant.value, constant.comment)
		}
		specs = append(specs, c)
	}
	if len(specs) == 0 {
		return nil
	}
	return &ast.GenDecl{Tok: token.CONST, Specs: specs}
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

func (g *Generator) writeFunctions() error {
	for _, name := range g.funcNames {
		if f, ok := g.funcValues[name]; ok {
			if b, err := f.String(); err != nil {
				return err
			} else {
				g.writeBody(b)
			}
		}
		g.writeBody("\n")
	}
	return nil
}

func (g *Generator) writeTypes() error {
	return g.writeSpecs(g.getTypes())
}

func (g *Generator) getTypes() *ast.GenDecl {
	specs := []ast.Spec{}
	names := g.typeNames
	values := g.typeValues
	for _, name := range names {
		value := values[name]
		specs = append(specs, newType(name, value))
	}
	if len(specs) == 0 {
		return nil
	}
	return &ast.GenDecl{Tok: token.TYPE, Specs: specs}
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

func (g *Generator) addConstDelim() {
	if len(g.constNames) > 0 {
		g.constNames = append(g.constNames, "")
	}
}

func (g *Generator) addFunсDelim() {
	if len(g.funcNames) > 0 {
		g.funcNames = append(g.funcNames, "")
	}
}

func (g *Generator) addConst(name, value, typ string) error {
	if exists, ok := g.constants[name]; ok && exists.value != value {
		return errors.Errorf("duplicated constant with different value; const %v, values: %v, %v", name, exists, value)
	}
	g.constNames = append(g.constNames, name)
	g.constants[name] = constant{value: value, typ: typ}
	return nil
}

func (g *Generator) addVarDelim() {
	if len(g.varNames) > 0 {
		g.varNames = append(g.varNames, "")
	}
}

func (g *Generator) AddFuncDecl(node *ast.FuncDecl) error {
	funcName := FuncDeclName(node)
	if exists, ok := g.funcValues[funcName]; ok {
		es, err := stringifyAst(exists.node)
		if err != nil {
			es = err.Error()
		}
		ns, err := stringifyAst(node)
		if err != nil {
			ns = err.Error()
		}
		if es != ns {
			return errors.Errorf("duplicated func with different value; func %s, declarations: %s, %s", funcName, es, ns)
		}
	}
	g.funcNames = append(g.funcNames, funcName)
	g.funcValues[funcName] = funcBody{node: node}
	return nil
}

func FuncDeclName(funcDecl *ast.FuncDecl) string {
	name := funcDecl.Name.Name
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		name = funcDecl.Recv.List[0].Names[0].Name + "." + name
	}
	return name
}

func stringifyAst(node ast.Node) (string, error) {
	fset := token.NewFileSet()
	out := &bytes.Buffer{}
	if node == nil {
		return "", nil
	}
	if err := printer.Fprint(out, fset, node); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (g *Generator) AddFunc(funcName, funcValue string) error {
	if exists, ok := g.funcValues[funcName]; ok && exists.body != funcValue {
		return errors.Errorf("duplicated func with different value; const %v, values: %v, %v", funcName, exists.body, funcValue)
	}
	g.funcNames = append(g.funcNames, funcName)
	g.funcValues[funcName] = funcBody{body: funcValue}
	return nil
}

func (g *Generator) AddImport(pack, alias string) {
	if exists, ok := g.imports[pack]; ok {
		logger.Debugf("replace imported package %s by %s, alias %s", exists, pack, alias)
	}
	g.imports[pack] = alias
}

func isFieldExcluded(fieldName struc.FieldName, includePrivate bool) bool {
	return (!includePrivate && !token.IsExported(string(fieldName)))
}

func renameFuncByConfig(funcName, renameTo string) string {
	if len(renameTo) > 0 {
		logger.Debugw("rename func %v to %v", funcName, renameTo)
		funcName = renameTo
	}
	return funcName
}

func AsRefIfNeed(receiverVar string, returnRefs bool) string {
	receiverRef := receiverVar
	if returnRefs {
		receiverRef = "&" + receiverRef
	}
	return receiverRef
}

func getTypeName(typeName string, pkg string) string {
	if len(pkg) > 0 {
		return pkg + "." + typeName
	}
	return typeName
}

func (g *Generator) getTagTemplateConstName(typeName string, fieldName struc.FieldName, tags []struc.TagName, export, snake bool) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	export = isExport(fieldName) && export
	tagsPart := ""
	for _, tag := range tags {
		tagsPart += getIdentPart(tag, snake)
	}
	return goName(typeName+tagsPart+getIdentPart(fieldName, snake), export)
}

func getUsedFieldConstName(typeName string, fieldName struc.FieldName, hardcodeValues, export, snake bool) string {
	if hardcodeValues {
		return quoted(fieldName)
	}
	return GetFieldConstName(typeName, fieldName, isExport(fieldName) && export, snake)
}

func convertFieldPathToGoIdent(fieldName struc.FieldName) string {
	return strings.ReplaceAll(fieldName, ".", "")
}

func (g *Generator) filterInjected() {
	g.typeNames = filterNotExisted(g.typeNames, g.typeValues)
	g.constNames = filterNotExisted(g.constNames, g.constants.nvMap())
	g.varNames = filterNotExisted(g.varNames, g.varValues)
	g.funcNames = filterNotExisted(g.funcNames, g.funcValues.names())

}

func noLint(nolint bool) string {
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

func GetFieldConstName(typeName string, fieldName struc.FieldName, export, snake bool) string {
	fieldName = convertFieldPathToGoIdent(fieldName)
	return goName(GetFieldType(typeName, export, snake)+getIdentPart(fieldName, snake), isExport(fieldName) && export)
}

func isExport(fieldName struc.FieldName) bool {
	return token.IsExported(fieldName)
}

func newComment(comment string) *ast.CommentGroup {
	if len(comment) == 0 {
		return nil
	}
	return &ast.CommentGroup{List: []*ast.Comment{{Text: comment}}}
}

func newConst(name, typ, val, comment string) *ast.ValueSpec {
	return &ast.ValueSpec{
		Names:   []*ast.Ident{{Name: name}},
		Type:    &ast.Ident{Name: typ},
		Values:  []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: val}},
		Comment: newComment(comment),
	}
}

func newType(name, typ string) *ast.TypeSpec {
	return &ast.TypeSpec{
		Name: &ast.Ident{Name: name},
		Type: &ast.Ident{Name: typ},
	}
}

func newImport(name, path string) *ast.ImportSpec {
	return &ast.ImportSpec{
		Name: &ast.Ident{Name: name},
		Path: &ast.BasicLit{Kind: token.STRING, Value: path},
	}
}

func quoted(value string) (constValue string) {
	return "\"" + value + "\""
}
