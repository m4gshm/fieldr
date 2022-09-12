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
	"unicode"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"

	"golang.org/x/tools/go/packages"
)

const oneLineSize = 3

type Generator struct {
	name string

	outFile      *ast.File
	outFileInfo  *token.File
	outPkg       *packages.Package
	outBuildTags string

	body *bytes.Buffer

	excludedTagValues map[string]bool
	excludedFields    map[struc.FieldName]interface{}

	rewrite CodeRewriter

	constNames    []string
	constValues   map[string]string
	constComments map[string]string
	varNames      []string
	varValues     map[string]string
	typeNames     []string
	typeValues    map[string]string
	funcNames     []string
	funcValues    map[string]string

	imports map[string]string

	isRewrite bool
}

func New(name, outBuildTags string, outFile *ast.File, outFileInfo *token.File, outPkg *packages.Package) *Generator {
	g := &Generator{
		name:              name,
		outBuildTags:      outBuildTags,
		outFile:           outFile,
		outFileInfo:       outFileInfo,
		outPkg:            outPkg,
		constNames:        make([]string, 0),
		constValues:       make(map[string]string),
		constComments:     make(map[string]string),
		varNames:          make([]string, 0),
		varValues:         make(map[string]string),
		typeNames:         make([]string, 0),
		typeValues:        make(map[string]string),
		funcNames:         make([]string, 0),
		funcValues:        make(map[string]string),
		imports:           map[string]string{},
		excludedTagValues: make(map[string]bool),
		excludedFields:    make(map[struc.FieldName]interface{}),
	}
	g.isRewrite = g.IsRewrite(outFile, outFileInfo)
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
	if g.isRewrite {
		g.body = &bytes.Buffer{}
		g.writeHead(outPackageName)
		g.writeTypes()
		g.writeConstants()
		g.writeVars()
		g.writeFunctions()
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

		g.writeTypes()
		g.writeConstants()
		g.writeVars()
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
							if generatingValues != nil {
								name := objectName.Name
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
			} else if funcDecl, hasFuncDecl := g.funcValues[name]; hasFuncDecl {
				chunks[start] = map[int]string{end: funcDecl}
				delete(g.funcValues, name)
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
	mName := MethodName(receiverName, name)
	if funcDecl, hasFuncDecl := g.funcValues[mName]; hasFuncDecl {
		chunks[start] = map[int]string{end: funcDecl}
		delete(g.funcValues, mName)
		return true, nil

	}
	return false, nil
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

func quoted(value interface{}) string {
	return "\"" + fmt.Sprintf("%v", value) + "\""
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

func (g *Generator) addConst(constName, constValue string) error {
	if exists, ok := g.constValues[constName]; ok && exists != constValue {
		return errors.Errorf("duplicated constant with different value; const %v, values: %v, %v", constName, exists, constValue)
	}
	g.constNames = append(g.constNames, constName)
	g.constValues[constName] = constValue
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

func (g *Generator) AddFunc(funcName, funcValue string) error {
	if exists, ok := g.funcValues[funcName]; ok && exists != funcValue {
		return errors.Errorf("duplicated func with different value; const %v, values: %v, %v", funcName, exists, funcValue)
	}
	g.funcNames = append(g.funcNames, funcName)
	g.funcValues[funcName] = funcValue
	return nil
}

func (g *Generator) AddImport(pack, alias string) {
	if exists, ok := g.imports[pack]; ok {
		logger.Debugf("replace imported package %s by %s, alias %s", exists, pack, alias)
	}
	g.imports[pack] = alias
}

func (g *Generator) isFieldExcluded(fieldName struc.FieldName, allFields bool) bool {
	_, excluded := g.excludedFields[fieldName]
	return isFieldExcluded(fieldName, allFields) || excluded
}

func isFieldExcluded(fieldName struc.FieldName, includePrivate bool) bool {
	return (!includePrivate && !token.IsExported(string(fieldName)))
}

func (g *Generator) Transform(fieldName struc.FieldName, fieldType struc.FieldType, fieldRef string) string {
	return g.rewrite.Transform(fieldName, fieldType, fieldRef)
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

func (g *Generator) getUsedFieldConstName(typeName string, fieldName struc.FieldName, hardcodeValues, export, snake bool) string {
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
	g.constNames = filterNotExisted(g.constNames, g.constValues)
	g.varNames = filterNotExisted(g.varNames, g.varValues)
	g.funcNames = filterNotExisted(g.funcNames, g.funcValues)

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
