package command

import (
	"flag"

	// "github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func NewStringifyEnum() *Command {
	const (
		name = "enrich-enum"
	)
	var (
		flagSet            = flag.NewFlagSet(name, flag.ExitOnError)
		toStringMethodName = flagSet.String("string-method", "String", "to string method converter name")
		valuesMethodName   = flagSet.String("values-func", generator.Autoname, "values function name, use "+generator.Autoname+" for autoname (<Type name>Values as default)")
		export             = params.Export(flagSet, true)
		nolint             = params.Nolint(flagSet)
	)
	c := New(
		name, "enriches an enum type with a convert to string method and a values enumeration function",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			m, err := context.EnumModel()
			if err != nil {
				return err
			}
			funcName, funcBody, err := g.GenerateEnumStringify(m, *toStringMethodName, *export, *nolint)
			if err != nil {
				return err
			}
			err = g.AddFuncOrMethod(funcName, funcBody)
			if err != nil {
				return err
			}
			funcName, funcBody, err = g.GenerateEnumValues(m, *valuesMethodName, *export, *nolint)
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
