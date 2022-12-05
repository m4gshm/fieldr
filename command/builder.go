package command

import (
	"flag"
	"fmt"
	"go/types"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
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
		flagSet         = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		name            = flagSet.String("name", generator.Autoname, "generated type name, use "+generator.Autoname+" for autoname")
		buildMethodName = flagSet.String("build-method-name", generator.Autoname, "generated build (constructor) method name, use "+generator.Autoname+" for autoname")
		setterPrefix    = flagSet.String("setter-prefix", generator.Autoname, "generated 'Set<Field>' methods prefix, use "+generator.Autoname+" for autoselect")
		chainValue      = flagSet.Bool("chain-value", false, "returns value of the builder in generated methods (default is pointer)")
		buildValue      = flagSet.Bool("build-value", false, "returns value of the builded type in the build (constructor) method (default is pointer)")
		light           = flagSet.Bool("light", false, "don't generate builder methods, only fields")
		exports         = params.MultiValFixed(flagSet, "export", []string{"methods"}, exportVals, "export generated content")
		nolint          = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates builder API of a structure type",
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

			builderBody := struc.TypeString(btyp, g.OutPkg.PkgPath) + " struct {" + generator.NoLint(*nolint) + "\n"

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
					return fmt.Errorf("unexpected value %s", e)
				}
			}

			constrMethodName := "Build"
			if len(*buildMethodName) > 0 && *buildMethodName != generator.Autoname {
				constrMethodName = *buildMethodName
			}
			constrMethodName = generator.LegalIdentName(generator.IdentName(constrMethodName, exportMethods))
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)

			rec := "b"
			logger.Debugf("constrMethodName %v", constrMethodName)
			constrMethodBody := "func (" + rec + " " + builderName + typeParams + ") " + constrMethodName + "() " + ifElse(*buildValue, "", "*") + buildedType + typeParams +
				" {" + generator.NoLint(*nolint) + "\n"
			constrMethodBody += "return " + ifElse(*buildValue, "", "&") + buildedType + typeParams + " {\n"

			uniques := map[string]string{}
			methodPrefix := ""
			if len(*setterPrefix) > 0 && (*setterPrefix) != generator.Autoname {
				methodPrefix = *setterPrefix
			}
			if exportFields == exportMethods && methodPrefix == "" {
				methodPrefix = defMethPref
			}

			c, b, fmn, fmb, err := generateBuilderParts(g, model, uniques, rec, ifElse(*chainValue, "", "*")+builderName+typeParams, methodPrefix, *light, exportMethods, exportFields, *nolint, *buildValue)
			if err != nil {
				return err
			}
			constrMethodBody += c
			builderBody += b

			builderBody += "}"
			constrMethodBody += "}\n}\n"

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

func ifElse[T any](condition bool, first, second T) T {
	if condition {
		return first
	}
	return second
}

func generateBuilderParts(
	g *generator.Generator, model *struc.Model, uniques map[string]string, builderRecVar, builderType, setterPrefix string, noMethods, exportMethods, exportFields, nolint, buildReceiver bool,
) (string, string, []string, []string, error) {
	logger.Debugf("generate builder parts: receiver %v, builderType %v, setterPrefix %v", builderRecVar, builderType, setterPrefix)
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
			c, b, fmn, fmb, err := generateBuilderParts(g, fieldType.Model, uniques, builderRecVar, builderType, setterPrefix, noMethods, exportMethods, exportFields, nolint, buildReceiver)
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
				fieldMethodName := generator.LegalIdentName(generator.IdentName(setterPrefix+builderField, exportMethods))
				arg := generator.LegalIdentName(generator.IdentName(builderField, false))

				fieldMethod := "func (" + builderRecVar + " " + builderType + ") " + fieldMethodName + "(" + arg + " " + fullFieldType + ") " + builderType +
					" {" + generator.NoLint(nolint) + "\n"
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
