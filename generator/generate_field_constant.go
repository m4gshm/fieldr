package generator

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/pkg/errors"
)

type stringer struct {
	val      string
	callback func()
}

func (c *stringer) String() string {
	if c == nil {
		return ""
	}
	c.callback()
	val := c.val
	return val
}

var _ fmt.Stringer = (*stringer)(nil)

func (g *Generator) GenerateFieldConstants(model *struc.Model, typ string, export, snake, allFields bool, flats map[string]struct{}) ([]fieldConst, error) {
	constants, err := makeFieldConsts(g, model, export, snake, allFields, flats)
	if err != nil {
		return nil, err
	} else if err := checkDuplicates(constants); err != nil {
		return nil, err
	}

	for _, c := range constants {
		name, val := c.name, Quoted(c.value)
		if err := g.addConst(name, val, typ); err != nil {
			return nil, err
		}
	}

	g.addConstDelim()

	return constants, nil
}

func (g *Generator) GenerateFieldConstant(
	model *struc.Model, valueTmpl, nameTmpl, typ, funcList string, export, snake, nolint, compact, usePrivate, refAccessor, valAccessor bool, flats map[string]struct{},
) error {
	valueTmpl, nameTmpl = wrapTemplate(valueTmpl), wrapTemplate(nameTmpl)

	wrapType := len(typ) > 0
	if !wrapType {
		typ = BaseConstType
	} else if err := g.AddType(typ, BaseConstType); err != nil {
		return err
	}

	constants, err := makeFieldConstsTempl(g, model, model.TypeName, nameTmpl, valueTmpl, export, snake, usePrivate, flats)
	if err != nil {
		return err
	} else if err = checkDuplicates(constants); err != nil {
		return err
	}
	for _, constant := range constants {
		if err := g.addConst(constant.name, Quoted(constant.value), typ); err != nil {
			return err
		}
	}
	g.addConstDelim()

	exportFunc := export
	if len(funcList) > 0 {
		funcName := funcList
		if funcName == Autoname {
			if wrapType {
				funcName = IdentName(typ+"s", export)
			} else {
				return fmt.Errorf("list function autoname is unsupported without constant type definition")
			}
		}
		if funcBody, funcName, err := generateAggregateFunc(funcName, typ, constants, exportFunc, compact, nolint); err != nil {
			return err
		} else if err := g.AddFunc(funcName, funcBody); err != nil {
			return err
		}
		g.addFunсDelim()
	}

	if wrapType {
		if funcBody, funcName, err := g.generateConstFieldMethod(typ, constants, exportFunc, nolint); err != nil {
			return err
		} else if err := g.AddMethod(typ, funcName, funcBody); err != nil {
			return err
		}
		g.addFunсDelim()

		if refAccessor || valAccessor {
			pkgAlias, err := g.GetPackageAlias(model.Package.Name, model.Package.Path)
			if err != nil {
				return err
			}
			if valAccessor {
				if funcBody, funcName, err := g.generateConstValueMethod(model, pkgAlias, typ, constants, exportFunc, nolint, false); err != nil {
					return err
				} else if err := g.AddMethod(typ, funcName, funcBody); err != nil {
					return err
				}
			}
			if refAccessor {
				if funcBody, funcName, err := g.generateConstValueMethod(model, pkgAlias, typ, constants, exportFunc, nolint, true); err != nil {
					return err
				} else if err := g.AddMethod(typ, funcName, funcBody); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type fieldInfo struct {
	name string
	typ  struc.FieldType
}
type fieldConst struct {
	name, value string
	fieldPath   []fieldInfo
}

func checkDuplicates(constants []fieldConst) error {
	uniqueVals, uniqueNames := map[string]string{}, map[string]string{}
	for _, c := range constants {
		name, value := c.name, c.value
		if dupl, ok := uniqueVals[value]; ok {
			return fmt.Errorf("duplicated constant values: first const '%s', second '%s', value '%s'", dupl, name, value)
		} else if dupl, ok := uniqueNames[name]; ok {
			return fmt.Errorf("duplicated constants: constant '%s', first value '%s', second '%s'", name, dupl, value)
		}
		uniqueVals[value], uniqueNames[name] = name, value
	}
	return nil
}

func makeFieldConstsTempl(g *Generator, model *struc.Model, structType, nameTmpl, valueTmpl string, export, snake, usePrivate bool, flats map[string]struct{}) ([]fieldConst, error) {
	usedTags := []string{}
	usedTagsSet := map[string]struct{}{}

	constants := make([]fieldConst, 0)

	if model == nil {
		return constants, nil
	}

	for _, f := range model.FieldNames {
		fieldName := f

		if isFieldExcluded(fieldName, usePrivate) {
			continue
		}

		fieldType := model.FieldsType[fieldName]
		embedded := fieldType.Embedded
		_, flat := flats[fieldName]
		fieldModel := fieldType.Model
		if flat || embedded {
			subflats := map[string]struct{}{}
			if embedded {
				subflats = flats
			}
			fieldConstants, err := makeFieldConstsTempl(g, fieldModel, structType, nameTmpl, valueTmpl, export, snake, usePrivate, subflats)
			if err != nil {
				return nil, err
			}
			for i := range fieldConstants {
				fieldConstants[i].fieldPath = append([]fieldInfo{{name: fieldName, typ: fieldType}}, fieldConstants[i].fieldPath...)
			}
			constants = append(constants, fieldConstants...)
		} else {
			tags := map[string]*stringer{}
			var inExecute bool
			if tagVals := model.FieldsTagValue[fieldName]; tagVals != nil {
				for k, v := range tagVals {
					tag := k
					tags[tag] = &stringer{val: v, callback: func() {
						if !inExecute {
							return
						}
						if _, ok := usedTagsSet[tag]; !ok {
							usedTagsSet[tag] = struct{}{}
							usedTags = append(usedTags, tag)
							logger.Debugf("use tag '%s'", tag)
						}
					}}
				}
			}

			parse := func(name string, data interface{}, funcs template.FuncMap, tmplVal string) (string, error) {
				logger.Debugf("parse template for \"%s\" %s\n", name, tmplVal)
				tmpl, err := template.New(name).Option("missingkey=zero").Funcs(funcs).Parse(tmplVal)
				if err != nil {
					return "", fmt.Errorf("parse: of '%s', template %s: %w", name, tmplVal, err)
				}

				buf := bytes.Buffer{}
				logger.Debugf("template context %+v\n", tags)
				inExecute = true
				if err = tmpl.Execute(&buf, data); err != nil {
					inExecute = false
					return "", fmt.Errorf("compile: of '%s': field '%s', template %s: %w", name, fieldName, tmplVal, err)
				}
				inExecute = false
				cmpVal := buf.String()
				logger.Debugf("parse result: of '%s'; %s\n", name, cmpVal)
				return cmpVal, nil
			}

			funcs := addCommonFuncs(template.FuncMap{
				"struct": func() map[string]interface{} { return map[string]interface{}{"name": structType} },
				"name":   func() string { return fieldName },
				"field":  func() map[string]interface{} { return map[string]interface{}{"name": fieldName} },
				"tag":    func() map[string]*stringer { return tags },
			})

			val, err := parse(fieldName+" const val", tags, funcs, valueTmpl)
			if err != nil {
				return nil, err
			}

			var constName string
			if len(nameTmpl) > 0 {
				parsedConst, err := parse(fieldName+" const name", tags, funcs, nameTmpl)
				if err != nil {
					return nil, err
				}
				constName = strings.ReplaceAll(parsedConst, ".", "")
			} else {
				constName = g.getTagTemplateConstName(structType, fieldName, usedTags, export, snake)
				logger.Debugf("apply auto constant name '%s'", constName)
			}

			if len(val) > 0 {
				constants = append(constants, fieldConst{
					name:      constName,
					value:     val,
					fieldPath: []fieldInfo{{name: fieldName, typ: fieldType}}})
			} else {
				logger.Infof("constant without value: '%s'", constName)
			}
		}
	}
	return constants, nil
}

func makeFieldConsts(g *Generator, model *struc.Model, export, snake, allFields bool, flats map[string]struct{}) ([]fieldConst, error) {
	constants := []fieldConst{}
	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		embedded := fieldType.Embedded
		_, flat := flats[fieldName]
		fieldModel := fieldType.Model
		filedInfo := fieldInfo{name: fieldName, typ: fieldType}
		if flat || embedded {
			subflats := map[string]struct{}{}
			if embedded {
				subflats = flats
			}
			fieldConstants, err := makeFieldConsts(g, fieldModel, export, snake, allFields, subflats)
			if err != nil {
				return nil, err
			}
			for i := range fieldConstants {
				fieldConstants[i].name = fieldName + fieldConstants[i].name
				fieldConstants[i].fieldPath = append([]fieldInfo{filedInfo}, fieldConstants[i].fieldPath...)
			}
			constants = append(constants, fieldConstants...)
		} else if allFields || isExport(fieldName) {
			constants = append(constants, fieldConst{
				name:      IdentName(fieldName, isExport(fieldName) && export),
				value:     fieldName,
				fieldPath: []fieldInfo{filedInfo}})
		}
	}
	return constants, nil
}

func addCommonFuncs(funcs template.FuncMap) template.FuncMap {
	toString := func(val interface{}) string {
		if val == nil {
			return ""
		}
		str := ""
		switch vt := val.(type) {
		case string:
			str = vt
		case fmt.Stringer:
			str = vt.String()
		case fmt.GoStringer:
			str = vt.GoString()
		default:
			str = fmt.Sprint(val)
			logger.Debugf("toString: val '%v', result '%s'", val, str)
		}
		return str
	}
	toStrings := func(vals []interface{}) []string {
		results := make([]string, len(vals))
		for i, val := range vals {
			results[i] = toString(val)
		}
		return results
	}
	rexp := func(expr interface{}, val interface{}) (string, error) {
		sexpr := toString(expr)
		str := toString(val)
		if len(sexpr) == 0 {
			return "", errors.New("empty regexp: val '" + str + "'")
		}
		r, err := regexp.Compile(sexpr)
		if err != nil {
			return "", err
		}
		submatches := r.FindStringSubmatch(str)
		names := r.SubexpNames()
		if len(names) <= len(submatches) {
			for i, groupName := range names {
				if groupName == "v" {
					submatch := submatches[i]
					if len(submatch) == 0 {
						return submatch, nil
					}
				}
			}
		}
		if len(submatches) > 0 {
			s := submatches[len(submatches)-1]
			return s, nil
		}
		return "", nil
	}

	snakeFunc := func(val interface{}) string {
		sval := toString(val)
		if len(sval) == 0 {
			return ""
		}
		last := len(sval) - 1
		symbols := []rune(sval)
		result := make([]rune, 0)
		for i := 0; i < len(symbols); i++ {
			cur := symbols[i]
			result = append(result, cur)
			if i < last {
				next := symbols[i+1]
				if unicode.IsLower(cur) && unicode.IsUpper(next) {
					result = append(result, '_', unicode.ToLower(next))
					i++
				}
			}
		}
		return string(result)
	}

	toUpper := func(val interface{}) string {
		return strings.ToUpper(toString(val))
	}

	toLower := func(val interface{}) string {
		return strings.ToLower(toString(val))
	}

	join := func(vals ...interface{}) string {
		result := strings.Join(toStrings(vals), "")
		return result
	}

	strOr := func(vals ...interface{}) string {
		if len(vals) == 0 {
			return ""
		}

		for _, val := range vals {
			s := toString(val)
			if len(s) > 0 {
				return s
			}
		}
		return toString(vals[len(vals)-1])
	}
	f := template.FuncMap{
		"OR":          strOr,
		"conc":        join,
		"concatenate": join,
		"join":        join,
		"regexp":      rexp,
		"rexp":        rexp,
		"snake":       snakeFunc,
		"toUpper":     toUpper,
		"toLower":     toLower,
		"up":          toUpper,
		"low":         toLower,
	}
	for k, v := range f {
		funcs[k] = v
	}
	return funcs
}

func wrapTemplate(text string) string {
	if len(text) == 0 {
		return text
	}
	if !strings.Contains(text, "{{") {
		text = "{{" + strings.ReplaceAll(text, "\\", "\\\\") + "}}"
		logger.Debugf("constant template transformed to '%s'", text)
	}
	return text
}

func generateAggregateFunc(funcName, typ string, constants []fieldConst, export, compact, nolint bool) (string, string, error) {
	var arrayType = "[]" + typ

	arrayBody := arrayType + "{"

	compact = compact || len(constants) <= oneLineSize
	if !compact {
		arrayBody += "\n"
	}

	i := 0
	for _, constant := range constants {
		if compact && i > 0 {
			arrayBody += ", "
		}
		arrayBody += constant.name
		if !compact {
			arrayBody += ",\n"
		}
		i++
	}
	arrayBody += "}"

	return "func " + funcName + "() " + arrayType + " { " + NoLint(nolint) + "\n return " + arrayBody + "}", funcName, nil
}

func (g *Generator) generateConstFieldMethod(typ string, constants []fieldConst, export, nolint bool) (string, string, error) {
	var (
		name         = IdentName("Field", export)
		receiverVar  = "c"
		returnType   = BaseConstType
		returnNoCase = "\"\""
	)

	body := "func (" + receiverVar + " " + typ + ") " + name + "() " + returnType
	body += " {" + NoLint(nolint) + "\n" +
		"switch " + receiverVar + " {\n" +
		""

	for _, constant := range constants {
		if len(constant.value) == 0 {
			continue
		}

		fieldPath := ""
		for _, p := range constant.fieldPath {
			if len(fieldPath) > 0 {
				fieldPath += "."
			}
			fieldPath += p.name
		}

		body += "case " + constant.name + ":\n" +
			"return \"" + fieldPath + "\"\n"
	}

	body += "}\n"
	body += "" +
		"return " + returnNoCase +
		"}\n"

	return body, name, nil
}

func (g *Generator) generateConstValueMethod(model *struc.Model, pkgAlias, typ string, constants []fieldConst, export, nolint, ref bool) (string, string, error) {
	var (
		name         = IdentName("Val", export)
		argVar       = "f"
		recVar       = "s"
		recType      = "*" + GetTypeName(model.TypeName, pkgAlias) + TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)
		returnTypes  = "interface{}"
		returnNoCase = "nil"
		pref         = ""
	)

	if ref {
		pref = "&"
		name = IdentName("Ref", export)
	}

	funSign := "func (" + recVar + " " + recType + ") " + name + "(" + argVar + " " + typ + ") " + returnTypes
	if len(pkgAlias) > 0 {
		funSign = "func " + name + "(" + recVar + " " + recType + ", " + argVar + " " + typ + ") " + returnTypes
	}
	body := funSign
	body += " {" + NoLint(nolint) + "\n"
	body += "if " + recVar + " == nil {\nreturn nil\n}\n"
	body += "switch " + argVar + " {\n" +
		""

	for _, constant := range constants {
		body += "case " + constant.name + ":\n"

		fullFieldPath, condition := FiledPathAndAcceddCheckCondition(recVar, constant.fieldPath)

		if len(condition) > 0 {
			body += "if " + condition + " {\n"
		}
		body += "return " + pref + fullFieldPath + "\n"
		if len(condition) > 0 {
			body += "}\n"
		}
	}

	body += "}\n"
	body += "" +
		"return " + returnNoCase +
		"}\n"

	return body, name, nil
}

func FiledPathAndAcceddCheckCondition(recVar string, fieldPathInfo []fieldInfo) (string, string) {
	condition := ""
	fieldPath := ""
	fullFieldPath := recVar + "."
	for _, p := range fieldPathInfo {
		if len(fieldPath) > 0 {
			fieldPath += "."
			fullFieldPath += "."
		}
		fieldPath += p.name
		fullFieldPath += p.name
		if p.typ.RefCount > 0 {
			if len(condition) > 0 {
				condition += " && "
			}
			condition += fullFieldPath + " != nil"
			for ri := 1; ri < p.typ.RefCount; ri++ {
				condition += " && "
				fullFieldPath = "*(" + fullFieldPath + ")"
				condition += fullFieldPath + " != nil"
				fullFieldPath = "(" + fullFieldPath + ")"
			}
		}
	}
	return fullFieldPath, condition
}
