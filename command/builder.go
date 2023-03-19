package command

import (
	"flag"
	"fmt"
	"go/types"
	"strings"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

func NewBuilderStruct() *Command {
	const (
		cmdName               = "builder"
		genContent            = "struct"
		defMethPref           = "Set"
		default_constructor   = "Build"
		default_deconstructor = "ToBuilder"
	)

	exportVals := []string{"all", "fields", "methods"}

	var (
		flagSet             = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		name                = flagSet.String("name", generator.Autoname, "generated type name, use "+generator.Autoname+" for autoname")
		buildMethodName     = flagSet.String("constructor", default_constructor, "generated constructor method name")
		setterPrefix        = flagSet.String("setter-prefix", generator.Autoname, "generated 'Set<Field>' methods prefix, use "+generator.Autoname+" for autoselect")
		chainValue          = flagSet.Bool("chain-value", false, "returns value of the builder in generated methods (default is pointer)")
		buildValue          = flagSet.Bool("build-value", false, "returns value of the builded type in the build (constructor) method (default is pointer)")
		light               = flagSet.Bool("light", false, "don't generate builder methods, only fields")
		toBuilderMethodName = flagSet.String("deconstructor", "", "generate instance to builder convert method, use "+generator.Autoname+" for autoname ("+default_deconstructor+")")
		exports             = params.MultiValFixed(flagSet, "export", []string{"methods"}, exportVals, "export generated content")
		nolint              = params.Nolint(flagSet)
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
			pkgName, err := g.GetPackageName(model.Package.Name, model.Package.Path)
			if err != nil {
				return err
			}
			buildedType := generator.GetTypeName(model.TypeName, pkgName)
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

			constrMethodName := default_constructor
			if len(*buildMethodName) > 0 && *buildMethodName != generator.Autoname {
				constrMethodName = *buildMethodName
			}
			constrMethodName = generator.LegalIdentName(generator.IdentName(constrMethodName, exportMethods))
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)

			receiver := "b"
			logger.Debugf("constrMethodName %v", constrMethodName)
			constrMethodBody := "func (" + receiver + " " + builderName + typeParams + ") " + constrMethodName + "() " + ifElse(*buildValue, "", "*") + buildedType + typeParams +
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

			builderType := ifElse(*chainValue, "", "*") + builderName + typeParams
			c, b, fmn, fmb, err := generateBuilderParts(g, model, uniques, receiver, builderType, methodPrefix, *light, exportMethods, exportFields, *nolint, *buildValue)
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

			if err := g.AddStruct(s); err != nil {
				return err
			}

			if len(*toBuilderMethodName) > 0 {
				*toBuilderMethodName = ifElse(*toBuilderMethodName == generator.Autoname, default_deconstructor, *toBuilderMethodName)
				builderType := ifElse(*buildValue, "", "*") + builderName + typeParams
				builderInstantiate := ifElse(*buildValue, "", "&") + builderName + typeParams
				instanceType := ifElse(*buildValue, "", "*") + buildedType + typeParams
				instanceReceiver := "i"
				toBuilderMethodBody := "func (" + instanceReceiver + " " + instanceType + ") " + *toBuilderMethodName + "() " + builderType +
					" {" + generator.NoLint(*nolint) + "\n"

				b, pre, err := generateToBuilderMethodParts(g, model, instanceReceiver, "", exportFields)
				if err != nil {
					return err
				}
				toBuilderMethodBody += pre
				toBuilderMethodBody += "return " + builderInstantiate + " {\n"
				toBuilderMethodBody += b + "\n"

				toBuilderMethodBody += "}\n}\n"
				return g.AddMethod(model.TypeName, *toBuilderMethodName, toBuilderMethodBody)
			}

			return nil
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
	g *generator.Generator, model *struc.Model, uniques map[string]string, receiverVar, typeName, setterPrefix string, noMethods, exportMethods, exportFields, nolint, buildReceiver bool,
) (string, string, []string, []string, error) {
	logger.Debugf("generate builder parts: receiver %v, type %v, setterPrefix %v", receiverVar, typeName, setterPrefix)
	constructorMethodBody := ""
	structBody := ""
	fieldMethodBodies := []string{}
	fieldMethodNames := []string{}
	for i, fieldName := range model.FieldNames {
		if i > 0 {
			structBody += "\n"
		}
		fieldType := model.FieldsType[fieldName]
		fullFieldType := fieldType.Name

		if fieldType.Embedded {
			t, _, err := struc.GetTypeNamed(fieldType.Type)
			if err != nil {
				return "", "", nil, nil, err
			}
			typeParams := ""
			if t != nil {
				typeParams = generator.TypeArgsString(t.TypeArgs(), g.OutPkg.PkgPath)
			}
			init := fullFieldType + typeParams
			if fieldType.RefCount > 0 {
				init = "&" + init
			}
			constructorMethodBody += fieldName + ": " + init + "{\n"
			c, b, fmn, fmb, err := generateBuilderParts(g, fieldType.Model, uniques, receiverVar, typeName, setterPrefix, noMethods, exportMethods, exportFields, nolint, buildReceiver)
			if err != nil {
				return "", "", nil, nil, err
			}
			constructorMethodBody += c
			structBody += b
			constructorMethodBody += "\n}"
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
			structBody += builderField + " " + fullFieldType
			constructorMethodBody += fieldName + ": " + receiverVar + "." + builderField
			if !noMethods {
				fieldMethodName := generator.LegalIdentName(generator.IdentName(setterPrefix+builderField, exportMethods))
				arg := generator.LegalIdentName(generator.IdentName(builderField, false))

				fieldMethod := "func (" + receiverVar + " " + typeName + ") " + fieldMethodName + "(" + arg + " " + fullFieldType + ") " + typeName +
					" {" + generator.NoLint(nolint) + "\n"
				fieldMethod += receiverVar + "." + builderField + "=" + arg + "\n"
				fieldMethod += "return " + receiverVar + "\n}\n"
				fieldMethodBodies = append(fieldMethodBodies, fieldMethod)
				fieldMethodNames = append(fieldMethodNames, fieldMethodName)
			}
		}
		constructorMethodBody += ",\n"
	}
	return constructorMethodBody, structBody, fieldMethodNames, fieldMethodBodies, nil
}

func generateToBuilderMethodParts(
	g *generator.Generator, model *struc.Model, receiver, fieldPrefix string, exportFields bool,
) (string, string, error) {
	logger.Debugf("generate toBuilder method: receiver %v", receiver)
	initVarsInitPart := ""
	methodBody := ""
	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]

		if fieldType.Embedded {
			fieldPathInfo := []generator.FieldInfo{{Name: fieldType.Name, Type: fieldType}}
			fullFieldPath, condition := generator.FiledPathAndAcceddCheckCondition(receiver, fieldPathInfo)
			if len(condition) > 0 {
				m, i, err := generateToBuilderMethodConditionedParts(fieldPathInfo, fieldType.Model, fullFieldPath, condition, receiver)
				if err != nil {
					return "", "", err
				}
				methodBody += m
				initVarsInitPart += i
			} else {
				c, _, err := generateToBuilderMethodParts(g, fieldType.Model, receiver, fieldType.Name, exportFields)
				if err != nil {
					return "", "", err
				}
				methodBody += c
			}

		} else {
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			methodBody += builderField + ": " + receiver + "." + ifElse(len(fieldPrefix) > 0, fieldPrefix+".", "") + fieldName
			methodBody += ",\n"
		}
	}
	return methodBody, ifElse(len(initVarsInitPart) > 0, initVarsInitPart+"\n", ""), nil
}

func generateToBuilderMethodConditionedParts(parentFieldPathInfo []generator.FieldInfo, model *struc.Model, fullFieldPath, condition, receiver string) (string, string, error) {
	initVarsInitPart := ""
	methodBody := ""

	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		handled := false
		if fieldType.Embedded {
			fieldPathInfo := append(append([]generator.FieldInfo{}, parentFieldPathInfo...), generator.FieldInfo{Name: fieldType.Name, Type: fieldType})
			fullFieldPath, subCondition := generator.FiledPathAndAcceddCheckCondition(receiver, fieldPathInfo)
			if len(subCondition) > 0 {
				m, i, err := generateToBuilderMethodConditionedParts(fieldPathInfo, fieldType.Model, fullFieldPath, subCondition, receiver)
				if err != nil {
					return "", "", err
				}
				methodBody += m
				initVarsInitPart += i
				handled = true
			}
		}

		if !handled {
			varPref := strings.ReplaceAll(fullFieldPath, ".", "_")
			varName := varPref + "_" + strings.ReplaceAll(fieldName, ".", "_")

			initVarsInitPart += "var " + varName + " " + fieldType.FullName + "\n"
			initVarsInitPart += "if " + condition + " {\n"
			initVarsInitPart += varName + "=" + fullFieldPath + "." + fieldName
			initVarsInitPart += "}\n"

			builderField := generator.LegalIdentName(generator.IdentName(fieldName, true))
			methodBody += builderField + ": " + varName
			methodBody += ",\n"

		}
	}

	return methodBody, initVarsInitPart, nil
}
