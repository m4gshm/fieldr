package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/struc"
)

func NewBuilderStruct() *Command {
	const (
		cmdName    = "builder"
		genContent = "struct"
	)

	var (
		flagSet = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		name    = flagSet.String("name", generator.Autoname, "generated type name, use "+generator.Autoname+" for autoname")
		// flat    = params.Flat(flagSet)
		// export  = params.Export(flagSet)
		// snake   = params.Snake(flagSet)
		// nolint  = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates structure that used as builder of any struct type of named arguments function caller",
		flagSet,
		func(context *Context) error {
			model, err := context.Model()
			if err != nil {
				return err
			}
			g := context.Generator
			pkgAlias, err := g.GetPackageAlias(model.Package.Name, model.Package.Path)
			if err != nil {
				return err
			}
			buildedType := generator.GetTypeName(model.TypeName, pkgAlias)
			builderName := model.TypeName + "Builder"

			if *name != generator.Autoname {
				builderName = *name
			}

			builderBody := "type " + builderName + " struct {\n"
			rec := "b"
			constrMethodName := "Build"
			constrMethodBody := "func (" + rec + " " + builderName + ") " + constrMethodName + "() *" + buildedType + " {\n" +
				"return &" + buildedType + " {\n"
			for i, fieldName := range model.FieldNames {
				if i > 0 {
					builderBody += "\n"
					constrMethodBody += ",\n"
				}
				fieldType := model.FieldsType[fieldName]
				fullFieldType := fieldType.Name
				if typ, err := g.Repack(fieldType.Type, model.Package.Name); err != nil {
					return err
				} else {
					fullFieldType = struc.TypeString(typ, model.Package.Name)
				}
				builderField := generator.IdentName(fieldName, true)
				builderBody += builderField + " " + fullFieldType
				constrMethodBody += fieldName + ": " + rec + "." + builderField
			}

			builderBody += "}"
			constrMethodBody += "}\n}"

			s := generator.Structure{Name: builderName, Body: builderBody}
			if err := s.AddMethod(constrMethodName, constrMethodBody); err != nil {
				return err
			}

			return g.AddStruct(s)
		},
	)
}
