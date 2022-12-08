package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"go/types"
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
	OutPkg       *packages.Package
	outBuildTags string

	body *bytes.Buffer

	constNames []string
	constants  constants

	varNames     []string
	varValues    map[string]string
	typeNames    []string
	typeValues   map[string]string
	structNames  []string
	structBodies map[string]string
	funcNames    []string
	funcBodies   funcBodies

	imports map[string]string

	rewriteOutFile bool
}

type funcBodies map[string]funcBody

type Structure struct {
	Name, Body  string
	methodNames []string
	methods     map[string]string
}

func (s *Structure) AddMethod(name, val string) error {
	if s.methods == nil {
		s.methods = map[string]string{}
	}
	if exists, ok := s.methods[name]; ok && exists != val {
		return errors.Errorf("duplicated method with different value; type %s, method %s, exists '%s', new '%s'", s.Name, name, exists, val)
	} else if !ok {
		s.methodNames = append(s.methodNames, name)
		s.methods[name] = val
	}
	return nil
}

func (c funcBodies) names() map[string]string {
	r := make(map[string]string, len(c))
	for k := range c {
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
	return &Generator{
		name:           name,
		outBuildTags:   outBuildTags,
		outFile:        outFile,
		outFileInfo:    outFileInfo,
		OutPkg:         outPkg,
		constNames:     []string{},
		constants:      map[string]constant{},
		varNames:       []string{},
		varValues:      map[string]string{},
		typeNames:      []string{},
		typeValues:     map[string]string{},
		structNames:    []string{},
		structBodies:   map[string]string{},
		funcNames:      []string{},
		funcBodies:     make(map[string]funcBody),
		imports:        map[string]string{},
		rewriteOutFile: isRewrite(outFile, outFileInfo, generatedMarker(name)),
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

func (g *Generator) GetPackageAlias(pkgName, pkgPath string) (string, error) {
	needImport := pkgPath != g.OutPkg.PkgPath
	if !needImport {
		return "", nil
	}
	if exists, ok := g.imports[pkgPath]; ok {
		return exists, nil
	}
	pkgAlias := pkgName
	if !g.rewriteOutFile {
		alias, found, err := g.findImportPackageAlias(pkgPath, g.outFile)
		if err != nil {
			return "", err
		}
		if found {
			needImport = false
			if len(alias) > 0 {
				pkgAlias = alias
			}
		} else {
			structPackageSuffixed := pkgAlias
			duplicated := false
			i := 0
			for i <= 100 {
				if duplicated, err = hasDuplicatedPackage(g.outFile, structPackageSuffixed); err != nil {
					return "", err
				} else if duplicated {
					i++
					structPackageSuffixed = pkgAlias + strconv.Itoa(i)
				} else {
					break
				}
			}
			if !duplicated && i > 0 {
				pkgAlias = structPackageSuffixed
			}
		}
	}

	if needImport {
		importAlias := pkgAlias
		if name := packagePathToName(pkgPath); name == pkgAlias {
			importAlias = ""
		}
		if _, err := g.AddImport(pkgPath, importAlias); err != nil {
			return "", err
		}
	}
	return pkgAlias, nil
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

		if err := g.writeStructs(); err != nil {
			return err
		}

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
		if err := g.writeStructs(); err != nil {
			return err
		}
		if err := g.writeFunctions(); err != nil {
			return err
		}
	}
	return nil
}

func isRewrite(outFile *ast.File, outFileInfo *token.File, generatedMarker string) bool {
	if outFile == nil {
		return true
	}
	for _, comment := range outFile.Comments {
		pos := comment.Pos()
		base := outFileInfo.Base()
		firstComment := int(pos) == base
		if firstComment {
			text := comment.Text()
			generated := strings.HasPrefix(text, generatedMarker)
			return generated
		}
	}
	return false
}

func (g *Generator) findImportPackageAlias(pkgPath string, outFile *ast.File) (string, bool, error) {
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
					} else if imported := value == pkgPath; imported {
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
				start := int(dt.Pos()) - base
				end := int(dt.End()) - base
				imports := g.getImports()
				if len(imports.Specs) == 0 {
					imports.Specs = []ast.Spec{}
				}
				imports.Specs = append(imports.Specs, dt.Specs...)
				out := &bytes.Buffer{}
				if err := writeSpecs(imports, out); err != nil {
					return nil, err
				}
				if out.Len() > 0 {
					out.WriteString("\n")
				}
				expr := out.String()
				if len(expr) > 0 {
					chunks[start] = map[int]string{end: expr}
				}
				importInjected = true
			} else {
				specs := dt.Specs
				for _, spec := range specs {
					struc := false
					switch st := spec.(type) {
					case *ast.TypeSpec:
						switch st.Type.(type) {
						case *ast.Ident, *ast.ArrayType:
						case *ast.StructType:
							struc = true
						default:
							continue
						}
						start := int(st.Type.Pos()) - base
						end := int(st.Type.End()) - base
						name := st.Name.Name

						if struc {
							start = int(st.Pos()) - base
							if newValue, found := g.structBodies[name]; found {
								chunks[start] = map[int]string{end: newValue}
								delete(g.structBodies, name)
							}
						} else {
							if newValue, found := g.typeValues[name]; found {
								chunks[start] = map[int]string{end: newValue}
								delete(g.typeValues, name)
							}
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
				if _, err := g.addReceiverFuncOnRewrite(recv.List, name, chunks, start, end); err != nil {
					return nil, err
				}
			} else if _, err := g.moveFuncToChunks(name, chunks, start, end); err != nil {
				return nil, err
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

func (g *Generator) addReceiverFuncOnRewrite(list []*ast.Field, name string, chunks map[int]map[int]string, start, end int) (bool, error) {
	if len(list) == 0 {
		return false, nil
	}
	field := list[0]
	receiverName, err := getReceiverName(field.Type)
	if err != nil {
		return false, fmt.Errorf("func %v; %w", name, err)
	}
	return g.moveFuncToChunks(MethodName(receiverName, name), chunks, start, end)
}

func (g *Generator) moveFuncToChunks(name string, chunks map[int]map[int]string, start, end int) (bool, error) {
	if f, ok := g.funcBodies[name]; !ok {
		return false, nil
	} else if b, err := f.String(); err != nil {
		return false, err
	} else {
		chunks[start] = map[int]string{end: b}
		delete(g.funcBodies, name)
		return true, nil
	}
}

func MethodName(typ, fun string) string { return typ + "." + fun }

func getReceiverName(typ ast.Expr) (string, error) {
	switch tt := typ.(type) {
	case *ast.StarExpr:
		return getReceiverName(tt.X)
	case *ast.Ident:
		return tt.Name, nil
	case *ast.IndexExpr:
		return getReceiverName(tt.X)
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
		return "", errors.Errorf("receiver name; unexpected type %v, value %v", reflect.TypeOf(tt), tt)
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
	g.writeBody("// %s'; DO NOT EDIT.\n\n", generatedMarker(g.name))
	if tags := strings.Trim(g.outBuildTags, " "); len(tags) > 0 {
		g.writeBody("// +build " + tags + "\n")
	}
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
	imports := g.getImports()
	if len(imports.Specs) == 0 {
		return "", nil
	}
	out := &bytes.Buffer{}
	if err := writeSpecs(imports, out); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (g *Generator) getImports() *ast.GenDecl {
	specs := []ast.Spec{}
	for pack, alias := range g.imports {
		specs = append(specs, newImport(alias, pack))
	}
	// if len(specs) == 0 {
	// 	return nil
	// }
	return &ast.GenDecl{Tok: token.IMPORT, Specs: specs}
}

func writeSpecs(specs ast.Node, out *bytes.Buffer) error {
	if specs == nil || (reflect.ValueOf(specs).Kind() == reflect.Ptr && reflect.ValueOf(specs).IsNil()) {
		return nil
	}
	fset := token.NewFileSet()
	if err := printer.Fprint(out, fset, specs); err != nil {
		return err
	}
	out.WriteString("\n")
	return nil
}

func (g *Generator) writeConstants() error {
	return writeSpecs(g.getConstants(), g.body)
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
		if f, ok := g.funcBodies[name]; ok {
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
	return writeSpecs(g.getTypes(), g.body)
}

func (g *Generator) writeStructs() error {
	for _, name := range g.structNames {
		if s, ok := g.structBodies[name]; ok {
			g.writeBody("type " + s)
		}
		g.writeBody("\n")
	}
	return nil
}

func (g *Generator) getTypes() *ast.GenDecl {
	specs := []ast.Spec{}
	for _, name := range g.typeNames {
		value := g.typeValues[name]
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

func generatedMarker(name string) string {
	return fmt.Sprintf("Code generated by '%s", name)
}

func GetFieldType(typeName string, export, snake bool) string {
	return LegalIdentName(IdentName(typeName+getIdentPart("Field", snake), export))
}

func getIdentPart(suffix string, snake bool) string {
	if snake {
		return "_" + suffix
	}
	return camel(suffix)
}

func IdentName(name string, export bool) string {
	first := rune(name[0])
	if export {
		first = unicode.ToUpper(first)
	} else {
		first = unicode.ToLower(first)
	}
	result := string(first) + name[1:]
	return result
}

var illegals = map[string]struct{}{
	"return": {},
	"type":   {},
	"chan":   {},
	"struct": {},
	"false":  {},
	"true":   {},
	"func":   {},
}

func LegalIdentName(name string) string {
	for {
		if _, ok := illegals[name]; !ok {
			return name
		}
		name = name + "_"
	}
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
		return errors.Errorf("duplicated constant with different value; const %s, exist '%s', new '%s'", name, exists, value)
	} else if !ok {
		g.constNames = append(g.constNames, name)
		g.constants[name] = constant{value: value, typ: typ}
	}
	return nil
}

func (g *Generator) addVarDelim() {
	if len(g.varNames) > 0 {
		g.varNames = append(g.varNames, "")
	}
}

func (g *Generator) AddFuncDecl(node *ast.FuncDecl) error {
	funcName := FuncDeclName(node)
	if exists, ok := g.funcBodies[funcName]; ok {
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
	} else if !ok {
		g.funcNames = append(g.funcNames, funcName)
		g.funcBodies[funcName] = funcBody{node: node}
	}
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

func (g *Generator) AddStruct(s Structure) error {
	typ := s.Name
	if exists, ok := g.structBodies[typ]; ok && exists != s.Body {
		return errors.Errorf("duplicated struct with different body; type %s, exist '%s', new '%s'", typ, exists, s.Body)
	} else if !ok {
		g.structNames = append(g.structNames, typ)
		g.structBodies[typ] = s.Body
		for _, name := range s.methodNames {
			if err := g.AddMethod(typ, name, s.methods[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *Generator) AddMethod(typ, name, body string) error {
	return g.AddFuncOrMethod(MethodName(typ, name), body)
}

func (g *Generator) AddFuncOrMethod(name, body string) error {
	if exists, ok := g.funcBodies[name]; ok && exists.body != body {
		return errors.Errorf("duplicated func with different value; func %s, exist '%s', new '%s'", name, exists.body, body)
	} else if !ok {
		g.funcNames = append(g.funcNames, name)
		g.funcBodies[name] = funcBody{body: body}
	}
	return nil
}

func (g *Generator) AddImport(pack, alias string) (string, error) {
	if len(pack) == 0 {
		return "", errors.New("empty package cannot be imported")
	}
	if exists, ok := g.imports[pack]; ok {
		if exists != alias {
			logger.Debugf("package alredy imported: package %s, exists alias %s, proposed %s", exists, pack, alias)
		}
		return exists, nil
	}
	g.imports[pack] = alias
	logger.Debugf("add import: package %s, alias %s", pack, alias)
	return alias, nil
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

func GetTypeName(typeName string, pkg string) string {
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
	return LegalIdentName(IdentName(typeName+tagsPart+getIdentPart(fieldName, snake), export))
}

func convertFieldPathToGoIdent(fieldName struc.FieldName) string {
	return strings.ReplaceAll(fieldName, ".", "")
}

func (g *Generator) filterInjected() {
	g.structNames = filterNotExisted(g.structNames, g.structBodies)
	g.typeNames = filterNotExisted(g.typeNames, g.typeValues)
	g.constNames = filterNotExisted(g.constNames, g.constants.nvMap())
	g.varNames = filterNotExisted(g.varNames, g.varValues)
	g.funcNames = filterNotExisted(g.funcNames, g.funcBodies.names())
}

func (g *Generator) ImportPack(pkg *types.Package, basePackagePath string) (*types.Package, error) {
	if pkg.Path() == basePackagePath {
		return pkg, nil
	}
	pkgPath := pkg.Path()
	if pkgAlias, err := g.AddImport(pkgPath, ""); err != nil {
		return nil, err
	} else if len(pkgAlias) > 0 {
		return types.NewPackage(pkgPath, pkgAlias), nil
	}
	return pkg, nil
}

func (g *Generator) RepackObj(typName *types.TypeName, basePackagePath string) (*types.TypeName, error) {
	pkg := typName.Pkg()
	if ipgk, err := g.ImportPack(pkg, basePackagePath); err != nil {
		return nil, err
	} else if pkg != ipgk {
		return types.NewTypeName(typName.Pos(), ipgk, typName.Name(), typName.Type()), nil
	}
	return typName, nil
}

func (g *Generator) RepackVar(vr *types.Var, basePackagePath string) (*types.Var, error) {
	pkg := vr.Pkg()
	if ipgk, err := g.ImportPack(pkg, basePackagePath); err != nil {
		return nil, err
	} else if pkg != ipgk {
		return types.NewVar(vr.Pos(), ipgk, vr.Name(), vr.Type()), nil
	}
	return vr, nil
}

func (g *Generator) RepackTuple(vr *types.Tuple, basePackagePath string) (*types.Tuple, error) {
	repacked := false
	r := make([]*types.Var, 0, vr.Len())
	for i := 0; i < vr.Len(); i++ {
		v := vr.At(i)
		rv, err := g.RepackVar(v, basePackagePath)
		if err != nil {
			return nil, err
		}
		repacked = repacked || rv != v
		r = append(r, rv)
	}

	if repacked {
		return types.NewTuple(r...), nil
	}
	return vr, nil
}

func (g *Generator) Repack(typ types.Type, basePackagePath string) (types.Type, error) {
	switch tt := typ.(type) {
	case *types.Named:
		obj := tt.Obj()
		if repaked, err := g.RepackObj(tt.Obj(), basePackagePath); err != nil {
			return nil, err
		} else if repaked != obj {
			methods := make([]*types.Func, tt.NumMethods())
			for i := range methods {
				methods[i] = tt.Method(i)
			}
			return types.NewNamed(repaked, tt.Underlying(), methods), nil
		}
	case *types.Pointer:
		e := tt.Elem()
		if re, err := g.Repack(e, basePackagePath); err != nil {
			return nil, err
		} else if re != e {
			return types.NewPointer(re), nil
		}
	case *types.Array:
		e := tt.Elem()
		if re, err := g.Repack(e, basePackagePath); err != nil {
			return nil, err
		} else if re != e {
			return types.NewArray(re, tt.Len()), nil
		}
	case *types.Slice:
		e := tt.Elem()
		if re, err := g.Repack(e, basePackagePath); err != nil {
			return nil, err
		} else if re != e {
			return types.NewSlice(re), nil
		}
	case *types.Chan:
		e := tt.Elem()
		if re, err := g.Repack(e, basePackagePath); err != nil {
			return nil, err
		} else if re != e {
			return types.NewChan(tt.Dir(), re), nil
		}
	case *types.Map:
		k := tt.Key()
		e := tt.Elem()
		if re, err := g.Repack(e, basePackagePath); err != nil {
			return nil, err
		} else {
			if rk, err := g.Repack(k, basePackagePath); err != nil {
				return nil, err
			} else if re != e || rk != k {
				return types.NewMap(rk, re), nil
			}
		}
	// case *types.Func:
	// 	pkg := tt.Pkg()
	// 	sign := tt.Type().(*types.Signature)
	// 	if rsign, err := g.Repack(sign, basePackagePath); err != nil {
	// 		return nil, err
	// 	} else if rsign != sign {
	// 		return types.NewFunc(tt.Pos(), pkg, tt.Name(), rsign), nil
	// 	}
	case *types.Signature:
		recv := tt.Recv()
		rrecv, err := g.RepackVar(recv, basePackagePath)
		if err != nil {
			return nil, err
		}
		rparams := tt.Params()
		rtuple, err := g.RepackTuple(rparams, basePackagePath)
		if err != nil {
			return nil, err
		}
		res := tt.Results()
		rres, err := g.RepackTuple(res, basePackagePath)
		if err != nil {
			return nil, err
		}
		if recv != rrecv || rparams != rtuple || res != rres {
			return types.NewSignature(rrecv, rparams, rres, tt.Variadic()), nil
		}
	}
	return typ, nil
}

func NoLint(nolint bool) string {
	if nolint {
		return "//nolint"
	}
	return ""
}

func filterNotExisted(names []string, values map[string]string) []string {
	newTypeNames := []string{}
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
	return LegalIdentName(IdentName(GetFieldType(typeName, export, snake)+getIdentPart(fieldName, snake), isExport(fieldName) && export))
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
		Path: &ast.BasicLit{Kind: token.STRING, Value: Quoted(path)},
	}
}

func Quoted(value string) string { return "\"" + value + "\"" }

func GetFieldRef(fields ...struc.FieldName) struc.FieldName {
	result := ""
	for _, field := range fields {
		if len(result) > 0 {
			result += "."
		}
		result += field
	}
	return result
}
