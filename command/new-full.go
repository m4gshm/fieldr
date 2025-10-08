package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/generator/constructor"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/gollections/collection/immutable"
	"github.com/m4gshm/gollections/predicate/is"
)

func NewNewFull() *Command {
	const (
		cmdName = "new-full"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		name            = flagSet.String("name", generator.Autoname, "constructor name, use "+generator.Autoname+" for autoname New<Type name>")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		returnVal       = flagSet.Bool("return-value", false, "returns value instead of pointer")
		flat            = flagSet.Bool("flat", false, "makes fields of emmbedded types constructor arguments")
		noinine         = flagSet.Bool("no-inline", false, "no inlines empty embedded structs")
		exclude         = params.MultiValFixed(flagSet, "exclude", nil, nil, "excluded argument")
		nolint          = params.Nolint(flagSet)
	)
	return New(
		cmdName, "generates a struct creation function with mandatory mapping of arguments to fields.",
		flagSet,
		func(context *Context) error {
			model, err := context.StructModel()
			if err != nil {
				return err
			}
			if name != nil && len(*name) > 0 {
				g := context.Generator
				isInclude := is.Not(immutable.NewSet(*exclude...).Contains)
				cname, body, err := constructor.FullArgs(g, model, *name, *returnVal, !(*noExportMethods), *nolint, *flat, *noinine, isInclude)
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
