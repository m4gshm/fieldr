package command

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

var counter = 0

func NewEnumConst() *Command {
	const (
		name     = "enum-const"
		flagVal  = "val"
		flagName = "name"
	)
	var (
		flagSet = flag.NewFlagSet(name, flag.ContinueOnError)

		constName  = flagSet.String("name", "", "constant name template")
		constValue = flagSet.String("val", "", "constant value template; must be set")
		export     = params.Export(flagSet, "constants")
		num        = counter
	)
	counter++
	c := New(
		name, "generate constants based on template applied to struct fields",
		flagSet,
		func(g *generator.Generator, m *struc.Model) error {
			return generateLookupConstant(g, m, *constValue, *constName, *export, false, false, num)
		},
	)
	c.manual =
		`Examples:
	` + name + ` -val \".json\" - use 'json' tag value as constant value, constant name is generated automatically, template corners '{{', '}}' can be omitted
	`
	return c

	// //te TagValues var per tag"),
	// 	EnumFieldConsts: multiVal(flagSet, enum_field_const, []string{}, "generate constants based on template applied to struct fields;"+
	// 	"\ntemplate examples:"+
	// 	"\n\t\".json\" - use 'json' tag value as constant value, constant name is generated automatically, template corners '{{', '}}' can be omitted"+
	// 	"\n\t\"{{name}}={{.json}}\" - use 'json' tag value as constant value, constant name based on field 'name', name/value delimeter '=' and template corners are '{{', '}}' required)"+
	// 	"\n\t\"{{(join struct.name field.name)| up}}={{tag.json}}\" - usage of functions 'join', 'up' and pipeline character '|' for more complex constant naming"+
	// 	"\n\t\"rexp tag.json \"(\\w+),?\" - regular expression."+
	// 	"\nfunctions:"+
	// 	"\n\tjoin, conc - strings concatenation; multiargs"+
	// 	"\n\tOR - select first non empty string argument; multiargs"+
	// 	"\n\trexp - find substring by regular expression; arg1: regular expression, arg2: string value; use 'v' group name as constant value marker, example: (?P<v>\\\\w+)"+
	// 	"\n\tup - convert string to upper case"+
	// 	"\n\tlow - convert string to lower case"+
	// 	"\n\tsnake - convert camel to snake case"+
	// 	"\nmetadata:"+
	// 	"\n\tname - current field name"+
	// 	"\n\tfield - current field metadata map"+
	// 	"\n\tstruct - struct type metadata map"+
	// 	"\n\ttag - tag names map"+
	// 	"\n\t.<tag name> - access to tag name"+
	// 	"",
}

type stringer struct {
	val      string
	callback func()
}

func (c *stringer) String() string {
	c.callback()
	return c.val
}

var _ fmt.Stringer = (*stringer)(nil)

func addCommonFuncs(funcs template.FuncMap) template.FuncMap {
	toString := func(val interface{}) string {
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
		if r, err := regexp.Compile(sexpr); err != nil {
			return "", err
		} else {
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
			} else {
				return "", nil
			}
		}
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

	strOr := func(vals ...string) string {
		if len(vals) == 0 {
			return ""
		}

		for _, val := range vals {
			if len(val) > 0 {
				return val
			}
		}
		return vals[len(vals)-1]
	}
	f := template.FuncMap{
		"OR":          strOr,
		"conc":        join,
		"concatenate": join,
		"join":        join,
		"regexp":      rexp,
		"rexp":        rexp,

		"snake":   snakeFunc,
		"toUpper": toUpper,
		"toLower": toLower,
		"up":      toUpper,
		"low":     toLower,
	}
	for k, v := range f {
		funcs[k] = v
	}
	return funcs
}

func generateLookupConstant(g *generator.Generator, model *struc.Model, value, name string, export, snake, wrapType bool, num int) error {
	wrapTmpl := func(text string) string {
		if len(text) == 0 {
			return text
		}
		if !strings.Contains(text, "{{") {
			text = "{{" + strings.ReplaceAll(text, "\\", "\\\\") + "}}"
			logger.Debugf("constant template transformed to '%s'", text)
		}
		return text
	}

	valueTmpl := wrapTmpl(value)
	nameTmpl := wrapTmpl(name)

	usedTags := []string{}
	usedTagsSet := map[string]struct{}{}

	type constResult struct{ name, autoName, field, value string }
	constants := make([]constResult, 0)
	for _, fieldName := range model.FieldNames {
		tags := map[string]interface{}{}
		if tagVals := model.FieldsTagValue[fieldName]; tagVals != nil {
			for k, v := range model.FieldsTagValue[fieldName] {
				tag := k
				tags[tag] = &stringer{val: v, callback: func() {
					if _, ok := usedTagsSet[tag]; !ok {
						usedTagsSet[tag] = struct{}{}
						usedTags = append(usedTags, tag)
					}
				}}
			}
		}

		parse := func(tmplVal string) (string, error) {
			funcs := addCommonFuncs(template.FuncMap{
				"struct": func() map[string]interface{} { return map[string]interface{}{"name": model.TypeName} },
				"name":   func() string { return fieldName },
				"field":  func() map[string]interface{} { return map[string]interface{}{"name": fieldName} },
				"tag":    func() map[string]interface{} { return tags },
			})

			logger.Debugf("parse template %s\n", tmplVal)
			tmpl, err := template.New(value).Option("missingkey=zero").Funcs(funcs).Parse(tmplVal)
			if err != nil {
				return "", fmt.Errorf("const lookup parse: template=%s: %w", tmplVal, err)
			}

			buf := bytes.Buffer{}
			logger.Debugf("execute template context %+v\n", tags)
			if err = tmpl.Execute(&buf, tags); err != nil {
				return "", fmt.Errorf("const lookup compile: field=%s, template='%s': %w", fieldName, tmplVal, err)
			}
			cmpVal := buf.String()
			return cmpVal, nil
		}

		if val, err := parse(valueTmpl); err != nil {
			return err
		} else if len(nameTmpl) > 0 {
			if constName, err := parse(nameTmpl); err != nil {
				return err
			} else {
				constants = append(constants, constResult{name: constName, value: val})
			}
		} else {
			constants = append(constants, constResult{autoName: fmt.Sprintf("lookup%d", num), field: fieldName, value: val})
		}
	}

	for _, c := range constants {
		constName := c.name
		if len(constName) == 0 {
			constName = g.GetTagTemplateConstName(model.TypeName, c.field, usedTags, export, snake)
		}
		if len(c.value) != 0 {

			if err := g.AddConst(constName, g.GetConstValue(generator.BaseConstType, c.value, wrapType)); err != nil {
				return err
			}
		}
	}

	return nil
}
