package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/generator/constructor"
	"github.com/m4gshm/fieldr/params"
)

func NewNewFull() *Command {
	const (
		cmdName = "new-full"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		name            = flagSet.String("name", generator.Autoname, "constructor name, use "+generator.Autoname+" for autoname New<Type name>")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		nolint          = params.Nolint(flagSet)
	)
	return New(
		cmdName, "generates a structure constructor in which all fields are required",
		flagSet,
		func(context *Context) error {
			model, err := context.StructModel()
			if err != nil {
				return err
			}
			if name != nil && len(*name) > 0 {
				g := context.Generator
				cname, body, err := constructor.FullArgs(g, model, *name, !(*noExportMethods), *nolint)
				if err != nil {
					return err
				} else if err := g.AddFuncOrMethod(cname, body); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
