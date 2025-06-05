package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func NewConstructFull() *Command {
	const (
		cmdName = "constructor"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		constructorName = flagSet.String("name", generator.Autoname, "constructor function name, use "+generator.Autoname+" for autoname (New<Type name> as default)")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		nolint          = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates a structure constructor thati initialize all fields",
		flagSet,
		func(context *Context) error {
			model, err := context.StructModel()
			if err != nil {
				return err
			}
			g := context.Generator

			initPart := ""
			args := ""
			for fieldName, fieldType := range model.FieldsNameAndType {
				args += fieldName + " " + fieldType.FullName(g.OutPkgPath) + ",\n"
				initPart += fieldName + ":" + fieldName + ",\n"
			}
			if len(initPart) > 0 {
				initPart = "\n" + initPart
			}
			if len(args) > 0 {
				args = "\n" + args
			}

			typeParams := TypeParamsString(model, g)
			typeParamsDecl := TypeParamsDeclarationString(model, g)

			constrName, constructorBody := GenerateConstructor(*constructorName, model.TypeName(), typeParamsDecl, typeParams, !(*noExportMethods), *nolint, args, initPart, nil)
			g.AddFuncOrMethod(constrName, constructorBody)
			return nil
		},
	)
}
