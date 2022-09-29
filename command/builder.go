package command

import (
	"flag"
	"fmt"
	"go/types"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

func NewBuilderStruct() *Command {
	const (
		cmdName     = "builder"
		genContent  = "struct"
		defMethPref = "Set"
	)

	exportVals := []string{"all", "fields", "methods"}

	var (
		flagSet      = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		name         = flagSet.String("name", generator.Autoname, "generated type name, use "+generator.Autoname+" for autoname")
		methodPrefix = flagSet.String("method-prefix", generator.Autoname, "generated methods prefix, use "+generator.Autoname+" for autoselect")
		ligth        = flagSet.Bool("ligth", false, "don't generate builder methods, only fields")
		exports      = params.MultiValFixed(flagSet, "export", []string{}, exportVals, "export generated content")
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

			var exportMethods, exportFields bool
			for _, e := range *exports {
				switch e {
				case "all":
					exportMethods = true
					exportFields = true
				case "methods":
					exportMethods = true
				case "fields":
					exportFields = true
				default:
					return fmt.Errorf("unsupported ")
				}
			}
			uniques := map[string]string{}
			prefix := ""
			if len(*methodPrefix) > 0 && (*methodPrefix) != generator.Autoname {
				prefix = *methodPrefix
			}
			if exportFields == exportMethods && prefix == "" {
				prefix = defMethPref
			}
			c, b, fmn, fmb, err := generateBuilderParts(g, model, uniques, rec, builderName+typeParams, *ligth, exportMethods, exportFields, prefix)
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

			for i := range fmn {
				fieldMethodName := fmn[i]
				fieldMethodBody := fmb[i]
				if err := s.AddMethod(fieldMethodName, fieldMethodBody); err != nil {
					return err
				}
			}
			return g.AddStruct(s)
		},
	)
}

func generateBuilderParts(
	g *generator.Generator, model *struc.Model, uniques map[string]string, builderRecVar, builderType string, noMethods, exportMethods, exportFields bool, prefix string,
) (string, string, []string, []string, error) {

	constrMethodBody := ""
	builderBody := ""
	fieldMethodBodies := []string{}
	fieldMethodNames := []string{}
	for i, fieldName := range model.FieldNames {
		if i > 0 {
			builderBody += "\n"
		}
		fieldType := model.FieldsType[fieldName]
		fullFieldType := fieldType.Name

		if fieldType.Embedded {
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)
			init := fullFieldType + typeParams
			if fieldType.RefCount > 0 {
				init = "&" + init
			}
			constrMethodBody += fieldName + ": " + init + "{\n"
			c, b, fmn, fmb, err := generateBuilderParts(g, fieldType.Model, uniques, builderRecVar, builderType, noMethods, exportMethods, exportFields, prefix)
			if err != nil {
				return "", "", nil, nil, err
			}
			constrMethodBody += c
			builderBody += b
			constrMethodBody += "\n}"
			if !noMethods {
				fieldMethodBodies = append(fieldMethodBodies, fmb...)
				fieldMethodNames = append(fieldMethodNames, fmn...)
			}
		} else {
			if typ, err := g.Repack(fieldType.Type, g.OutPkg.PkgPath); err != nil {
				return "", "", nil, nil, err
			} else {
				fullFieldType = struc.TypeString(typ, g.OutPkg.PkgPath)
			}
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			if dupl, ok := uniques[builderField]; ok {
				return "", "", nil, nil, fmt.Errorf("duplicated builder fields: name '%s', first type '%s', second '%s'", builderField, dupl, fullFieldType)
			}
			uniques[builderField] = fullFieldType
			builderBody += builderField + " " + fullFieldType
			constrMethodBody += fieldName + ": " + builderRecVar + "." + builderField
			if !noMethods {
				fieldMethodName := generator.LegalIdentName(generator.IdentName(prefix+builderField, exportMethods))
				arg := "a"
				fieldMethod := "func (" + builderRecVar + " " + builderType + ") " + fieldMethodName + "(" + arg + " " + fullFieldType + ") " + builderType + " {\n"
				fieldMethod += builderRecVar + "." + builderField + "=" + arg + "\n"
				fieldMethod += "return " + builderRecVar + "\n}\n"
				fieldMethodBodies = append(fieldMethodBodies, fieldMethod)
				fieldMethodNames = append(fieldMethodNames, fieldMethodName)
			}
		}
		constrMethodBody += ",\n"
	}
	return constrMethodBody, builderBody, fieldMethodNames, fieldMethodBodies, nil
}
