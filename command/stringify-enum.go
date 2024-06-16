package command

import (
	"flag"
	"go/types"

	"github.com/m4gshm/flag/flagenum"
	"github.com/m4gshm/gollections/collection/immutable"
	ordermap "github.com/m4gshm/gollections/collection/immutable/ordered/map_"
	"github.com/m4gshm/gollections/slice"
	"github.com/m4gshm/gollections/slice/group"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func toString[F ~string](from F) string { return string(from) }
func fromString[F ~string](s string) F  { return F(s) }

func NewStringifyEnum() *Command {
	const (
		name = "stringify-enum"
	)
	type apiMethod string
	const (
		to_string   apiMethod = "to-string"
		from_string           = "from-string"
		values                = "enum-values"
	)

	var (
		defaultApis          = slice.Of(to_string, from_string, values)
		flagSet              = flag.NewFlagSet(name, flag.ExitOnError)
		toStringMethodName   = flagSet.String("to-string-meth", "String", "to string method converter name")
		fromStringMethodName = flagSet.String("from-string-func", generator.Autoname, "from string function name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixFromString+" as default)")
		valuesMethodName     = flagSet.String("values-func", generator.Autoname, "values function name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixValues+" as default)")
		apis, err            = flagenum.Multiple(flagSet, "api", defaultApis, defaultApis, fromString[apiMethod], toString[apiMethod], "generated api method or functions")
		export               = params.Export(flagSet)
		nolint               = params.Nolint(flagSet)
	)
	if err != nil {
		panic(err)
	}

	selectedApis := immutable.NewSet(*apis...)

	return New(
		name, "enriches an enum type with a convert to string method, a convert string to the enum value funxtion and a values enumeration function",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			model, err := context.EnumModel()
			if err != nil {
				return err
			}
			constValNamesMap := ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
			typ := model.Typ()
			if selectedApis.Contains(to_string) {
				funcName, funcBody, err := g.GenerateEnumStringify(typ, constValNamesMap, *toStringMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(values) {
				funcName, funcBody, err := g.GenerateEnumValues(typ, constValNamesMap, *valuesMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(from_string) {
				funcName, funcBody, err := g.GenerateEnumFromString(typ, constValNamesMap.Values(), *fromStringMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
