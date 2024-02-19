package command

import (
	"flag"

	"github.com/m4gshm/gollections/collection/immutable/set"

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
		flagSet            = flag.NewFlagSet(name, flag.ExitOnError)
		constName          = flagSet.String("name", "", "constant name expression")
		constValue         = flagSet.String("val", "", "constant value expression; must be set")
		constType          = flagSet.String("type", "", "constant type name")
		notDeclateConsType = flagSet.Bool("not-declare-type", false, "don't generate constant type declaration")
		fieldNameAccess    = flagSet.String("field-name-access", "", "add a method that returns the associated struct field name, use "+generator.Autoname+" for autoname")
		refAccessor        = flagSet.String("ref-access", "", "add a function or method that returns a reference to the struct field for each generated constant, use "+generator.Autoname+" for autoname")
		valAccessor        = flagSet.String("val-access", "", "add a function or method that returns a value to the struct field for each generated constant, use "+generator.Autoname+" for autoname")
		funcList           = flagSet.String("list", "", "generate function that return list of all generated constant values, use "+generator.Autoname+" for autoname")
		compact            = flagSet.Bool("compact", false, "generate single line code in aggregate functions, constants")
		export             = params.ExportCont(flagSet, "constants")
		private            = params.WithPrivate(flagSet)
		nolint             = params.Nolint(flagSet)
		flat               = params.Flat(flagSet)
		excluded           = params.MultiVal(flagSet, "exclude", []string{}, "excluded field name")
		include            = flagSet.String("include", "", "An expression that determines whether the field is used to create constants")
		uniqueValues       = flagSet.Bool("check-unique-val", false, "checks if generated constant values are unique")
	)
	c := New(
		name, "generate constants based on expressions applied to struct fields",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			m, err := context.Model()
			if err != nil {
				return err
			}
			return g.GenerateFieldConstant(
				m, *constValue, *constName, *constType, *funcList, *fieldNameAccess, *refAccessor, *valAccessor, *export, false, *nolint, *compact, *private, *notDeclateConsType, *uniqueValues,
				set.New(*flat), set.New(*excluded), *include,
			)
		},
	)
	c.manual =
		`Examples:
	` + name + ` -` + flagVal + ` tag.json - using 'json' tag value as constant value, constant name is generated automatically.
	` + name + ` -` + flagName + ` 'name' -` + flagVal + ` 'tag.json' - the same as the previous one, but constant name is based on field's name.
	` + name + ` -` + flagVal + ` 'tag.json' -include 'tag.json != nil' - same as the previous one, but only includes filled tags.
	` + name + ` -` + flagVal + ` 'rexp("(\w+),?", tag.json)' - using 'regexp' function to extract json property name as constant value with removed ',omitempty' option.
	` + name + ` -` + flagName + ` 'struct.name + field.name | up()' -` + flagVal + ` 'tag.json' - concatenates type name with field name and converts it to uppercase using 'up' function"
Main functions:
	join, conc - strings concatenation; multiargs
	OR - select first non empty string argument; multiargs
	rexp - find substring by regular expression; arg1: regular expression, arg2: string value; use 'v' group name as constant value marker, example: (?P<v>\\w+)
	up - convert string to upper case
	low - convert string to lower case
	snake - convert camel to snake case
Metadata access:
	name, field.name - current field name
	field.type - current field type
	struct.type - struct type name
	tag.<tag name> - access to tag name
More info about expressions definition can be found here https://expr-lang.org/docs/language-definition`

	return c
}
