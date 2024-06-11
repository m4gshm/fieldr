package command

import (
	"flag"
	"go/types"

	ordermap "github.com/m4gshm/gollections/collection/immutable/ordered/map_"
	"github.com/m4gshm/gollections/slice/group"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func NewStringifyEnum() *Command {
	const (
		name = "enrich-enum"
	)
	var (
		flagSet              = flag.NewFlagSet(name, flag.ExitOnError)
		toStringMethodName   = flagSet.String("to-string-meth", "String", "to string method converter name")
		fromStringMethodName = flagSet.String("from-string-func", generator.Autoname, "from string function name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixFromString+" as default)")
		valuesMethodName     = flagSet.String("values-func", generator.Autoname, "values function name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixValues+" as default)")
		export               = params.Export(flagSet, true)
		nolint               = params.Nolint(flagSet)
	)
	c := New(
		name, "enriches an enum type with a convert to string method and a values enumeration function",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			model, err := context.EnumModel()
			if err != nil {
				return err
			}

			constValNamesMap := ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
			typ := model.Typ()

			funcName, funcBody, err := g.GenerateEnumStringify(typ, constValNamesMap, *toStringMethodName, *export, *nolint)
			if err != nil {
				return err
			}
			err = g.AddFuncOrMethod(funcName, funcBody)
			if err != nil {
				return err
			}
			funcName, funcBody, err = g.GenerateEnumValues(typ, constValNamesMap, *valuesMethodName, *export, *nolint)
			if err != nil {
				return err
			}
			err = g.AddFuncOrMethod(funcName, funcBody)
			if err != nil {
				return err
			}
			funcName, funcBody, err = g.GenerateEnumFromString(typ, constValNamesMap, *fromStringMethodName, *export, *nolint)
			if err != nil {
				return err
			}
			err = g.AddFuncOrMethod(funcName, funcBody)
			if err != nil {
				return err
			}
			return nil
		},
	)
	return c
}
