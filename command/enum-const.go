package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func NewEnumConst() *Command {
	const (
		name     = "enum-const"
		flagVal  = "val"
		flagName = "name"
	)
	var (
		flagSet     = flag.NewFlagSet(name, flag.ContinueOnError)
		constName   = flagSet.String("name", "", "constant name template")
		constValue  = flagSet.String("val", "", "constant value template; must be set")
		constType   = flagSet.String("type", "", "constant type name")
		refAccessor = flagSet.Bool("ref-access", false, "extends generated type with field reference accessor method")
		valAccessor = flagSet.Bool("val-access", false, "extends generated type with field value accessor method")
		funcList    = flagSet.String("list", "", "generate function that return list of all generated constant values, use "+generator.Autoname+" for autoname")
		compact     = flagSet.Bool("compact", false, "generate single line code in aggregate functions, constants")
		export      = params.ExportCont(flagSet, "constants")
		private     = params.WithPrivate(flagSet)
		nolint      = params.Nolint(flagSet)
		flat        = params.Flat(flagSet)
	)
	c := New(
		name, "generate constants based on template applied to struct fields",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			m, err := context.Model()
			if err != nil {
				return err
			}
			return g.GenerateFieldConstant(
				m, *constValue, *constName, *constType, *funcList, *export, false, *nolint, *compact, *private, *refAccessor, *valAccessor, toSet(*flat),
			)
		},
	)
	c.manual =
		`Examples:
	` + name + ` -` + flagVal + ` .json - usage of 'json' tag value as constant value, constant name is generated automatically, template corners '{{', '}}' can be omitted
	` + name + ` -` + flagName + ` '{{name}}' -` + flagVal + ` '{{.json}}' - the same as the previous one, but constant name is based on field's name
	` + name + ` -` + flagVal + ` 'rexp "(\w+),?" .json' - usage regexp function to extract json property name as constant value with removed ',omitempty' option
	` + name + ` -` + flagName + ` '{{(join struct.name field.name)| up}}' -` + flagVal + ` '{{tag.json}}' - usage of functions 'join', 'up' and pipeline character '|' for more complex constant naming"
Template functions:
	join, conc - strings concatenation; multiargs
	OR - select first non empty string argument; multiargs
	rexp - find substring by regular expression; arg1: regular expression, arg2: string value; use 'v' group name as constant value marker, example: (?P<v>\\\\w+)
	up - convert string to upper case
	low - convert string to lower case
	snake - convert camel to snake case
Metadata access:
	name - current field name
	field - current field metadata map
	struct - struct type metadata map
	tag - tag names map
	t.<tag name> - access to tag name`

	return c
}
