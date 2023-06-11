package generator

import (
	"bytes"
	"fmt"
	"go/token"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/collection/immutable"
	"github.com/m4gshm/gollections/collection/mutable/ordered"
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/replace"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/delay/string_/wrap"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/slice/convert"
	"github.com/m4gshm/gollections/slice/iter"
	"github.com/m4gshm/gollections/slice/split"
	"github.com/pkg/errors"
	"github.com/m4gshm/gollections/c"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
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

func (g *Generator) GenerateFieldConstants(model *struc.Model, typ string, export, snake, allFields bool, flats c.Checkable[string]) ([]FieldConst, error) {
	constants, err := makeFieldConsts(g, model, export, snake, allFields, flats)
	if err != nil {
		return nil, err
	} else if err := checkDuplicates(constants, true); err != nil {
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
	model *struc.Model, valueTmpl, nameTmpl, typ, funcList, typeMethod, refAccessor, valAccessor string,
	export, snake, nolint, compact, usePrivate, notDeclateConsType, uniqueValues bool,
	flats, excludedFields c.Checkable[string],
) error {
	valueTmpl, nameTmpl = wrapTemplate(valueTmpl), wrapTemplate(nameTmpl)

	wrapType := len(typ) > 0
	if !wrapType {
		typ = BaseConstType
	} else if !notDeclateConsType {
		if err := g.AddType(typ, BaseConstType); err != nil {
			return err
		}
	}

	logger.Debugf("GenerateFieldConstant wrapType %v, typ %v, nameTmpl %v valueTmpl %v\n", wrapType, typ, nameTmpl, valueTmpl)

	constants, err := makeFieldConstsTempl(g, model, model.TypeName, nameTmpl, valueTmpl, export, snake, usePrivate, flats, excludedFields)
	if err != nil {
		return err
	} else if err = checkDuplicates(constants, uniqueValues); err != nil {
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
		if funcName, err := use.If(funcList != Autoname, funcList).If(wrapType, IdentName(typ+"s", export)).ElseErr(
			fmt.Errorf("list function autoname is unsupported without constant type definition"),
		); err != nil {
			return err
		} else if funcBody, funcName, err := generateAggregateFunc(funcName, typ, constants, exportFunc, compact, nolint); err != nil {
			return err
		} else if err := g.AddFuncOrMethod(funcName, funcBody); err != nil {
			return err
		}
		g.addFunсDelim()
	}

	if wrapType {
		if len(typeMethod) > 0 {
			funcName := op.IfElse(typeMethod == Autoname, IdentName("Field", export), typeMethod)
			if funcBody, err := g.generateConstFieldMethod(typ, funcName, constants, exportFunc, nolint); err != nil {
				return err
			} else if err := g.AddMethod(typ, funcName, funcBody); err != nil {
				return err
			}
			g.addFunсDelim()
		}

		logger.Debugf("valAccessor %s, refAccessor %s", valAccessor, refAccessor)

		if len(refAccessor) != 0 || len(valAccessor) != 0 {
			pkgName, err := g.GetPackageName(model.Package.Name, model.Package.Path)
			if err != nil {
				return err
			}
			if len(valAccessor) != 0 {
				funcName := op.IfElse(valAccessor == Autoname, IdentName("Val", export), valAccessor)
				logger.Debugf("valAccessor func %s", funcName)
				if funcBody, funcName, err := g.generateConstValueMethod(model, pkgName, typ, funcName, constants, exportFunc, nolint, false); err != nil {
					return err
				} else if err := g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if len(refAccessor) != 0 {
				funcName := op.IfElse(refAccessor == Autoname, IdentName("Ref", export), refAccessor)
				logger.Debugf("refAccessor func %s", funcName)
				if funcBody, funcName, err := g.generateConstValueMethod(model, pkgName, typ, funcName, constants, exportFunc, nolint, true); err != nil {
					return err
				} else if err := g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type FieldConst struct {
	name, value string
	fieldPath   []FieldInfo
}

func (constant FieldConst) Name() string { return constant.name }

func checkDuplicates(constants []FieldConst, checkValues bool) error {
	uniqueVals, uniqueNames := map[string]string{}, map[string]string{}
	for _, c := range constants {
		name, value := c.name, c.value
		if dupl, ok := uniqueNames[name]; ok {
			return fmt.Errorf("duplicated constants: constant '%s', first value '%s', second '%s'", name, dupl, value)
		} else {
			uniqueNames[name] = value
		}
		if checkValues {
			if dupl, ok := uniqueVals[value]; ok {
				return fmt.Errorf("duplicated constant values: first const '%s', second '%s', value '%s'", dupl, name, value)
			} else {
				uniqueVals[value] = name
			}
		}
	}
	return nil
}

func makeFieldConstsTempl(
	g *Generator, model *struc.Model, structType, nameTmpl, valueTmpl string, export, snake, usePrivate bool, flats, excludedFields c.Checkable[string],
) ([]FieldConst, error) {
	var (
		usedTags  = &ordered.Set[struc.TagName]{}
		constants = make([]FieldConst, 0)
	)
	if model == nil {
		return constants, nil
	}

	for _, fieldName := range model.FieldNames {
		if !usePrivate && !token.IsExported(string(fieldName)) {
			logger.Debugf("exclude private field %v\n", fieldName)
			continue
		}

		if excludedFields.Contains(fieldName) {
			logger.Debugf("optional exclude field %v\n", fieldName)
			continue
		}

		fieldType := model.FieldsType[fieldName]
		embedded := fieldType.Embedded
		flat := flats.Contains(fieldName)
		fieldModel := fieldType.Model
		if flat || embedded {
			subflats := use.If(embedded, flats).Else(immutable.Set[string]{})
			fieldConstants, err := makeFieldConstsTempl(g, fieldModel, structType, nameTmpl, valueTmpl, export, snake, usePrivate, subflats, excludedFields)
			if err != nil {
				return nil, err
			}
			for i := range fieldConstants {
				fieldConstants[i].fieldPath = append([]FieldInfo{{Name: fieldName, Type: fieldType}}, fieldConstants[i].fieldPath...)
			}
			constants = append(constants, fieldConstants...)
		} else {
			var (
				tags      = map[string]*stringer{}
				inExecute bool
			)
			if tagVals := model.FieldsTagValue[fieldName]; tagVals != nil {
				for k, v := range tagVals {
					tag := k
					tags[tag] = &stringer{val: v, callback: func() {
						if !inExecute {
							return
						}
						if ok := usedTags.AddOneNew(tag); !ok {
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
				"field":  func() map[string]interface{} { return map[string]interface{}{"name": fieldName, "type": fieldType} },
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
				constName = g.getTagTemplateConstName(structType, fieldName, usedTags.Slice(), export, snake)
				logger.Debugf("apply auto constant name '%s'", constName)
			}

			if len(val) > 0 {
				constants = append(constants, FieldConst{
					name:      constName,
					value:     val,
					fieldPath: []FieldInfo{{Name: fieldName, Type: fieldType}}})
			} else {
				logger.Infof("constant without value: '%s'", constName)
			}
		}
	}
	return constants, nil
}

func makeFieldConsts(g *Generator, model *struc.Model, export, snake, allFields bool, flats c.Checkable[string]) ([]FieldConst, error) {
	constants := []FieldConst{}
	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		embedded := fieldType.Embedded
		flat := flats.Contains(fieldName)
		filedInfo := FieldInfo{Name: fieldName, Type: fieldType}
		if flat || embedded {
			fieldModel := fieldType.Model
			if fieldConstants, err := makeFieldConsts(g, fieldModel, export, snake, allFields,
				use.If(embedded, flats).Else(immutable.Set[string]{}),
			); err != nil {
				return nil, err
			} else {
				for i := range fieldConstants {
					fieldConstants[i].name = fieldName + fieldConstants[i].name
					fieldConstants[i].fieldPath = append([]FieldInfo{filedInfo}, fieldConstants[i].fieldPath...)
				}
				constants = append(constants, fieldConstants...)
			}
		} else if allFields || isExport(fieldName) {
			constants = append(constants, FieldConst{
				name:      IdentName(fieldName, isExport(fieldName) && export),
				value:     fieldName,
				fieldPath: []FieldInfo{filedInfo}})
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

func generateAggregateFunc(funcName, typ string, constants []FieldConst, export, compact, nolint bool) (string, string, error) {
	compact = compact || len(constants) <= oneLineSize
	var arrayType = "[]" + typ
	return "func " + funcName + "() " + arrayType + " { " + NoLint(nolint) + "\n return " + arrayType + "{" + op.IfElse(!compact, "\n", "") +
		convert.AndReduce(constants, FieldConst.Name, func(l, r string) string {
			return l + op.IfElse(len(l) > 0, op.IfElse(compact, ", ", ",\n"), "") + r
		}) + "}" + "}", funcName, nil
}

func (g *Generator) generateConstFieldMethod(typ, name string, constants []FieldConst, export, nolint bool) (string, error) {
	var (
		receiverVar  = "c"
		returnType   = BaseConstType
		returnNoCase = "\"\""
	)
	return "func (" + receiverVar + " " + typ + ") " + name + "() " + returnType + " {" + NoLint(nolint) + "\n" +
		"switch " + receiverVar + " {\n" + loop.Sum(iter.Convert(constants, func(constant FieldConst) string {
		return use.If(len(constant.value) == 0, "").ElseGet(sum.Of("case ", constant.name, ":\nreturn \"",
			convert.AndReduce(constant.fieldPath, func(p FieldInfo) string { return p.Name }, join.NonEmpty(".")), "\"\n"))
	}).Next) + "}\n" + "return " + returnNoCase + "}\n", nil
}

func (g *Generator) generateConstValueMethod(model *struc.Model, pkgName, typ, name string, constants []FieldConst, export, nolint, ref bool) (string, string, error) {
	var (
		argVar          = "f"
		recVar          = "s"
		recType         = GetTypeName(model.TypeName, pkgName)
		recParamType    = recType + TypeParamsString(model.Typ.TypeParams(), g.OutPkgPath)
		recParamTypeRef = "*" + recParamType
		returnTypes     = "interface{}"
		returnNoCase    = "nil"
		pref            = op.IfElse(ref, "&", "")
		isFunc          = len(pkgName) > 0
	)

	body := "func " + get.If(isFunc,
		sum.Of(name, "(", recVar, " ", recParamTypeRef, ", ", argVar, " ", typ, ") "),
	).ElseGet(
		sum.Of("(", recVar, " ", recParamTypeRef, ") ", name, "(", argVar, " ", typ, ") "),
	) + returnTypes + " {" + NoLint(nolint) + "\n" +
		"if " + recVar + " == nil {\nreturn nil\n}\n" +
		"switch " + argVar + " {\n" +
		loop.Reduce(iter.Convert(constants, func(constant FieldConst) string {
			_, conditionPath, conditions := FiledPathAndAccessCheckCondition(recVar, false, false, constant.fieldPath)
			varsConditionStart, varsConditionEnd := split.AndReduce(conditions, wrap.By("if ", " {\n"), replace.By("}\n"), op.Sum[string], op.Sum[string])
			return "case " + constant.name + ":\n" + varsConditionStart + "return " + pref + conditionPath + "\n" + varsConditionEnd
		}).Next, op.Sum[string]) +
		"}\nreturn " + returnNoCase + "}\n"

	return body, use.If(isFunc, name).Else(MethodName(recType, name)), nil
}
