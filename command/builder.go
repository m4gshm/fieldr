package command

import (
	"flag"
	"fmt"
	"go/types"

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

			typ := model.Typ
			obj := typ.Obj()

			btyp := types.NewNamed(
				types.NewTypeName(obj.Pos(), obj.Pkg(), builderName, types.NewStruct(nil, nil)), typ.Underlying(), nil,
			)

			tparams := typ.TypeParams()
			btparams := make([]*types.TypeParam, tparams.Len())
			for i := range btparams {
				tp := tparams.At(i)
				btparams[i] = types.NewTypeParam(tp.Obj(), tp.Constraint())
			}

			btyp.SetTypeParams(btparams)

			builderBody := "type " + struc.TypeString(btyp, g.OutPkg.PkgPath) + " struct {\n"
			rec := "b"
			constrMethodName := "Build"
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)
			constrMethodBody := "func (" + rec + " " + builderName + typeParams + ") " + constrMethodName + "() *" + buildedType + typeParams + " {\n" +
				"return &" + buildedType + typeParams + " {\n"
			c, b, err := generateBuilderParts(g, model, rec)
			if err != nil {
				return err
			}
			constrMethodBody += c
			builderBody += b

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

func generateBuilderParts(g *generator.Generator, model *struc.Model, rec string) (string, string, error) {
	uniques := map[string]string{}
	constrMethodBody := ""
	builderBody := ""
	for i, fieldName := range model.FieldNames {
		if i > 0 {
			builderBody += "\n"
		}
		fieldType := model.FieldsType[fieldName]
		fullFieldType := fieldType.Name

		if fieldType.Embedded {
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)
			init := fullFieldType + typeParams
			if fieldType.Ref {
				init = "&" + init
			}
			constrMethodBody += fieldName + ": " + init + "{\n"
			c, b, err := generateBuilderParts(g, fieldType.Model, rec)
			if err != nil {
				return "", "", err
			}
			constrMethodBody += c
			builderBody += b
			constrMethodBody += "\n}"
		} else {
			if typ, err := g.Repack(fieldType.Type, g.OutPkg.PkgPath); err != nil {
				return "", "", err
			} else {
				fullFieldType = struc.TypeString(typ, g.OutPkg.PkgPath)
			}
			builderField := generator.IdentName(fieldName, true)
			if dupl, ok := uniques[builderField]; ok {
				return "", "", fmt.Errorf("duplicated builder fields: name '%s', first type '%s', second '%s'", builderField, dupl, fullFieldType)
			}
			uniques[builderField] = fullFieldType
			builderBody += builderField + " " + fullFieldType
			constrMethodBody += fieldName + ": " + rec + "." + builderField
		}
		constrMethodBody += ",\n"
	}
	return constrMethodBody, builderBody, nil
}
