package command

import (
	"flag"

	"github.com/m4gshm/fieldr/params"
)

func NewStringifyEnum() *Command {
	const (
		name = "stringify-enum"
	)
	var (
		flagSet  = flag.NewFlagSet("stringify-enum", flag.ExitOnError)
		funcName = flagSet.String("name", "String", "function/method name")
		export   = params.Export(flagSet, true)
		nolint   = params.Nolint(flagSet)
	)
	c := New(
		name, "generate String() method for an enum type",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			m, err := context.EnumModel()
			if err != nil {
				return err
			}
			funcName, funcBody, err := g.GenerateEnumStringify(m, *funcName, *export, *nolint)
			if err != nil {
				return err
			}
			return g.AddFuncOrMethod(funcName, funcBody)
		},
	)
	return c
}
