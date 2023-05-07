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
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/use"
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
			builderName := op.IfElse(*name != generator.Autoname, *name, model.TypeName+"Builder")

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

			autogen := len(*buildMethodName) == 0 || *buildMethodName == generator.Autoname
			constrMethodName := generator.LegalIdentName(generator.IdentName(op.IfElse(autogen, default_constructor, *buildMethodName), exportMethods))
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

			builderType := op.IfElse(*chainValue, "", "*") + builderName + typeParams

			c, b, fmn, fmb, err := generateBuilderParts(g, model, uniques, receiver, builderType, methodPrefix,
				!(*chainValue), *light, exportMethods, exportFields, *nolint)
			if err != nil {
				return err
			}

			builderConstructorMethodName := op.IfElse(*newBuilderMethodName == generator.Autoname, "New"+builderName, *newBuilderMethodName)
			typeParamsDecl := generator.TypeParamsDeclarationString(model.Typ.TypeParams(), g.OutPkgPath)
			builderConstructorMethodBody := "func " + builderConstructorMethodName + typeParamsDecl + "() " + "*" + builderName + typeParams + "{\nreturn " +
				"&" + builderName + typeParams + "{}\n}\n"
			instanceConstructorMethodBody := "func (" + receiver + " " + builderType + ") " + constrMethodName + "() " +
				op.IfElse(*buildValue, "", "*") + buildedType + typeParams +
				" {" + generator.NoLint(*nolint) + "\n" +
				op.IfElse(*chainValue, "", "if "+receiver+" == nil {\n"+"return "+op.IfElse(*buildValue, "", "&")+buildedType+typeParams+" {}\n"+"}\n") +
				"return " + op.IfElse(*buildValue, "", "&") + buildedType + typeParams + " {\n" + c + "}\n" +
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
				*toBuilderMethodName = op.IfElse(*toBuilderMethodName == generator.Autoname, default_deconstructor, *toBuilderMethodName)
				builderType := op.IfElse(*chainValue, "", "*") + builderName + typeParams
				builderInstantiate := op.IfElse(*chainValue, "", "&") + builderName + typeParams
				instanceType := op.IfElse(*buildValue, "", "*") + buildedType + typeParams
				instanceReceiver := generator.TypeReceiverVar(model.TypeName)

				b, pre, err := generateToBuilderMethodParts(g, model, instanceReceiver, "", !(*buildValue), exportFields)
				if err != nil {
					return err
				}
				return g.AddMethod(model.TypeName, *toBuilderMethodName, "func "+use.If(len(pkgName) > 0,
					*toBuilderMethodName+typeParamsDecl+"("+instanceReceiver+" "+instanceType+") ",
				).Else(
					"("+instanceReceiver+" "+instanceType+") "+*toBuilderMethodName+"() ",
				)+builderType+" {"+generator.NoLint(*nolint)+"\n"+use.If(!*buildValue,
					"if "+instanceReceiver+" == nil {\nreturn "+builderInstantiate+" {}\n"+"}\n",
				).Else("")+pre+"return "+builderInstantiate+" {\n"+b+"\n"+"}\n}\n")
			}

			return nil
		},
	)
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
				init = op.IfElse(fieldType.RefCount > 0, "&"+fullFieldType, fullFieldType)
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

				fieldMethod += use.If(isReceiverReference, "if "+receiverVar+" != nil {\n").Else("") +
					receiverVar + "." + builderField + "=" + arg + "\n" + op.IfElse(isReceiverReference, "}\n", "") +
					"return " + receiverVar + "\n}\n"
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
					declVars := op.IfElse(len(vars) > 0, "var (\n"+strings.Join(vars, "\n")+"\n)\n", "")
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
			methodBody += builderField + ": " + receiver + "." + op.IfElse(len(fieldPrefix) > 0, fieldPrefix+".", "") + fieldName
			methodBody += ",\n"
		}
	}
	return methodBody, op.IfElse(len(varsPart) > 0, varsPart+"\n", ""), nil
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
			fullFielPath := parentPath + op.IfElse(len(fieldPath) > 0, "."+fieldPath, "")
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
