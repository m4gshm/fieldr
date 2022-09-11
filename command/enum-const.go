package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

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
		constType  = flagSet.String("type", "", "constant type template")
		export     = params.ExportCont(flagSet, "constants")
		nolint     = params.Nolint(flagSet)
		flat       = params.MultiVal(flagSet, "flat", []string{}, "apply generator to fields of nested structs")
	)
	c := New(
		name, "generate constants based on template applied to struct fields",
		flagSet,
		func(g *generator.Generator, m *struc.HierarchicalModel) error {
			model := toFlatModel(m, *flat)
			return generator.GenerateFieldConstant(g, model, *constValue, *constName, *constType, *export, false, *nolint)
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
