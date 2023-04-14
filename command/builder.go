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
		flagSet              = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		name                 = flagSet.String("name", generator.Autoname, "builder type name, use "+generator.Autoname+" for autoname (<Type name>Builder as default)")
		newBuilderMethodName = flagSet.String("method", generator.Autoname, "builder constructor method name, use "+generator.Autoname+" for autoname (New<Type name> as default)")
		buildMethodName      = flagSet.String("constructor", default_constructor, "target Type constructor method name")
		setterPrefix         = flagSet.String("setter-prefix", generator.Autoname, "setters methods prefix, use "+generator.Autoname+" for autoselect ('Set<Field>' as default)")
		chainValue           = flagSet.Bool("chain-value", false, "returns value of the builder in generated methods (default is pointer)")
		buildValue           = flagSet.Bool("build-value", false, "returns value of the builded type in the build (constructor) method (default is pointer)")
		light                = flagSet.Bool("light", false, "don't generate builder constructor and setters, only fields")
		toBuilderMethodName  = flagSet.String("deconstructor", "", "generate instance to builder convert method, use "+generator.Autoname+" for autoname ("+default_deconstructor+")")
		exports              = params.MultiValFixed(flagSet, "export", []string{"methods"}, exportVals, "export generated content")
		nolint               = params.Nolint(flagSet)
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
				types.NewTypeName(obj.Pos(), g.OutPkgTypes, builderName, types.NewStruct(nil, nil)), typ.Underlying(), nil,
			)

			tparams := typ.TypeParams()
			btparams := make([]*types.TypeParam, tparams.Len())
			for i := range btparams {
				tp := tparams.At(i)
				btparams[i] = types.NewTypeParam(tp.Obj(), tp.Constraint())
			}

			btyp.SetTypeParams(btparams)

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
			typeParams := generator.TypeParamsString(model.Typ.TypeParams(), g.OutPkgPath)

			receiver := "b"
			logger.Debugf("constrMethodName %v", constrMethodName)

			uniques := map[string]string{}
			methodPrefix := ""
			if len(*setterPrefix) > 0 && (*setterPrefix) != generator.Autoname {
				methodPrefix = *setterPrefix
			}
			if exportFields == exportMethods && methodPrefix == "" {
				methodPrefix = defMethPref
			}

			builderType := ifElse(*chainValue, "", "*") + builderName + typeParams

			c, b, fmn, fmb, err := generateBuilderParts(g, model, uniques, receiver, builderType, methodPrefix,
				!(*chainValue), *light, exportMethods, exportFields, *nolint)
			if err != nil {
				return err
			}

			builderConstructorMethodName := ifElse(*newBuilderMethodName == generator.Autoname, "New"+builderName, *newBuilderMethodName)
			typeParamsDecl := generator.TypeParamsDeclarationString(model.Typ.TypeParams(), g.OutPkgPath)
			builderConstructorMethodBody := "func " + builderConstructorMethodName + typeParamsDecl + "() " + "*" + builderName + typeParams + "{\nreturn " +
				"&" + builderName + typeParams + "{}\n}\n"
			instanceConstructorMethodBody := "func (" + receiver + " " + builderType + ") " + constrMethodName + "() " +
				ifElse(*buildValue, "", "*") + buildedType + typeParams +
				" {" + generator.NoLint(*nolint) + "\n" +
				ifElse(*chainValue, "", "if "+receiver+" == nil {\n"+"return "+ifElse(*buildValue, "", "&")+buildedType+typeParams+" {}\n"+"}\n") +
				"return " + ifElse(*buildValue, "", "&") + buildedType + typeParams + " {\n" + c + "}\n" +
				"}\n"

			builderBody := struc.TypeString(btyp, g.OutPkgPath) + " struct {" + generator.NoLint(*nolint) + "\n" + b + "}"

			if !*light {
				if err := g.AddFuncOrMethod(builderConstructorMethodName, builderConstructorMethodBody); err != nil {
					return err
				}
			}
			s := generator.Structure{Name: builderName, Body: builderBody}
			if err := s.AddMethod(constrMethodName, instanceConstructorMethodBody); err != nil {
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
				builderType := ifElse(*chainValue, "", "*") + builderName + typeParams
				builderInstantiate := ifElse(*chainValue, "", "&") + builderName + typeParams
				instanceType := ifElse(*buildValue, "", "*") + buildedType + typeParams
				instanceReceiver := generator.TypeReceiverVar(model.TypeName)

				b, pre, err := generateToBuilderMethodParts(g, model, instanceReceiver, "", !(*buildValue), exportFields)
				if err != nil {
					return err
				}
				toBuilderBody := ifElse(len(pkgName) > 0,
					"func "+*toBuilderMethodName+typeParamsDecl+"("+instanceReceiver+" "+instanceType+") ",
					"func ("+instanceReceiver+" "+instanceType+") "+*toBuilderMethodName+"() ",
				) + builderType + " {" + generator.NoLint(*nolint) + "\n"

				if !(*buildValue) {
					//nil entity case
					toBuilderBody += "if " + instanceReceiver + " == nil {\n" +
						"return " + builderInstantiate + " {}\n" +
						"}\n"
				}

				toBuilderBody += pre +
					"return " + builderInstantiate + " {\n" +
					b + "\n" +
					"}\n}\n"
				return g.AddMethod(model.TypeName, *toBuilderMethodName, toBuilderBody)
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
	g *generator.Generator, model *struc.Model, uniques map[string]string, receiverVar, typeName, setterPrefix string,
	isReceiverReference, noMethods, exportMethods, exportFields, nolint bool,
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

		if fieldType.Embedded {
			init := ""
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, true)
			if err != nil {
				return "", "", nil, nil, err
			} else {
				init = ifElse(fieldType.RefCount > 0, "&"+fullFieldType, fullFieldType)
			}

			c, b, fmn, fmb, err := generateBuilderParts(g, fieldType.Model, uniques, receiverVar, typeName, setterPrefix,
				isReceiverReference, noMethods, exportMethods, exportFields, nolint)
			if err != nil {
				return "", "", nil, nil, err
			}
			constructorMethodBody += fieldName + ": " + init + "{\n" + c + "\n}"
			structBody += b
			if !noMethods {
				fieldMethodBodies = append(fieldMethodBodies, fmb...)
				fieldMethodNames = append(fieldMethodNames, fmn...)
			}
		} else {
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
			if err != nil {
				return "", "", nil, nil, err
			}
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			if dupl, ok := uniques[builderField]; ok {
				return "", "", nil, nil, fmt.Errorf("duplicated builder fields: name '%s', first type '%s', second '%s'", builderField, dupl, fullFieldType)
			}
			uniques[builderField] = fullFieldType
			constructorMethodBody += fieldName + ": " + receiverVar + "." + builderField
			structBody += builderField + " " + fullFieldType
			if !noMethods {
				fieldMethodName := generator.LegalIdentName(generator.IdentName(setterPrefix+builderField, exportMethods))
				arg := generator.LegalIdentName(generator.IdentName(builderField, false))

				fieldMethod := "func (" + receiverVar + " " + typeName + ") " + fieldMethodName + "(" + arg + " " + fullFieldType + ") " + typeName +
					" {" + generator.NoLint(nolint) + "\n"

				fieldMethod += ifElse(isReceiverReference, "if "+receiverVar+" != nil {\n", "") + receiverVar + "." + builderField + "=" + arg + "\n" + ifElse(isReceiverReference, "}\n", "")
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
	g *generator.Generator, model *struc.Model, receiver, fieldPrefix string, isReceiverReference, exportFields bool,
) (string, string, error) {
	logger.Debugf("generate toBuilder method: receiver %v", receiver)
	varsPart := ""
	methodBody := ""
	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		if fieldType.Embedded {
			fieldPath, conditionalPath, conditions := generator.FiledPathAndAccessCheckCondition(receiver, false, false, []generator.FieldInfo{{Name: fieldType.Name, Type: fieldType}})
			if len(conditions) > 0 {
				if embedMethodBodyPart, vars, initVars, err := generateToBuilderMethodConditionedParts(
					fieldType.Model, fieldPath, conditionalPath, conditions, receiver, isReceiverReference, exportFields,
				); err != nil {
					return "", "", err
				} else {
					methodBody += embedMethodBodyPart
					declVars := ifElse(len(vars) > 0, "var (\n"+strings.Join(vars, "\n")+"\n)\n", "")
					varsPart += declVars + initVars
				}
			} else {
				if methodBodyPart, _, err := generateToBuilderMethodParts(
					g, fieldType.Model, receiver, fieldType.Name, isReceiverReference, exportFields,
				); err != nil {
					return "", "", err
				} else {
					methodBody += methodBodyPart
				}
			}
		} else {
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			methodBody += builderField + ": " + receiver + "." + ifElse(len(fieldPrefix) > 0, fieldPrefix+".", "") + fieldName
			methodBody += ",\n"
		}
	}
	return methodBody, ifElse(len(varsPart) > 0, varsPart+"\n", ""), nil
}

func generateToBuilderMethodConditionedParts(
	model *struc.Model, parentPath, conditionalPath string, conditions []string, receiver string, isReceiverReference, exportFields bool,
) (string, []string, string, error) {
	initVars := ""
	variables := []string{}
	methodBody := ""

	varsConditionStart := ""
	varsConditionEnd := ""
	for _, c := range conditions {
		varsConditionStart += "if " + c + " {\n"
		varsConditionEnd += "}\n"
	}

	initVars += varsConditionStart

	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		if fieldType.Embedded {
			fieldPath, conditionalPath, subConditions := generator.FiledPathAndAccessCheckCondition(conditionalPath, false, false, []generator.FieldInfo{{Name: fieldType.Name, Type: fieldType}})
			fullFielPath := parentPath + ifElse(len(fieldPath) > 0, "."+fieldPath, "")
			if m, embedVars, i, err := generateToBuilderMethodConditionedParts(
				fieldType.Model, fullFielPath, conditionalPath, subConditions, receiver, isReceiverReference, exportFields,
			); err != nil {
				return "", nil, "", err
			} else {
				methodBody += m
				variables = append(variables, embedVars...)
				initVars += i
			}
		} else {
			varName := generator.PathToVarName(parentPath + "." + fieldName)

			variables = append(variables, varName+" "+fieldType.FullName)
			initVars += varName + "=" + conditionalPath + "." + fieldName + "\n"

			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			methodBody += builderField + ": " + varName
			methodBody += ",\n"
		}
	}

	initVars += varsConditionEnd

	return methodBody, variables, initVars, nil
}
