package command

import (
	"flag"
	"fmt"
	"go/types"
	"strings"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/loop/convert"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/replace"
	"github.com/m4gshm/gollections/op/delay/string_/wrap"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/slice/split"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/fieldr/params"
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
		flagSet              = flag.NewFlagSet(cmdName, flag.ExitOnError)
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
			model, err := context.StructModel()
			if err != nil {
				return err
			}
			g := context.Generator
			pkgName, err := g.GetPackageNameOrAlias(model.Package().Name(), model.Package().Path())
			if err != nil {
				return err
			}
			typeName := model.TypeName()
			buildedType := generator.GetTypeName(typeName, pkgName)
			builderName := use.If(*name != generator.Autoname, *name).ElseGet(sum.Of(typeName, "Builder"))

			typ := model.Typ
			obj := typ.Obj()

			btyp := types.NewNamed(
				types.NewTypeName(obj.Pos(), g.OutPkgTypes, builderName, types.NewStruct(nil, nil)), typ.Underlying(), nil,
			)

			tparams := typ.TypeParams()

			btyp.SetTypeParams(loop.Slice(convert.FromIndexed(tparams.Len(), tparams.At, func(tp *types.TypeParam) *types.TypeParam {
				return types.NewTypeParam(tp.Obj(), tp.Constraint())
			})))

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

			parts, err := generateBuilderParts(g, model, uniques, receiver, builderType, methodPrefix,
				!(*chainValue), *light, exportMethods, exportFields, *nolint)
			if err != nil {
				return err
			}

			builderConstructorMethodName := get.If(*newBuilderMethodName == generator.Autoname, sum.Of("New", builderName)).Else(*newBuilderMethodName)
			typeParamsDecl := generator.TypeParamsDeclarationString(model.Typ.TypeParams(), g.OutPkgPath)
			builderConstructorMethodBody := "func " + builderConstructorMethodName + typeParamsDecl + "() " + "*" + builderName + typeParams + "{\nreturn " +
				"&" + builderName + typeParams + "{}\n}\n"
			instanceConstructorMethodBody := "func (" + receiver + " " + builderType + ") " + constrMethodName + "() " +
				op.IfElse(*buildValue, "", "*") + buildedType + typeParams +
				" {" + generator.NoLint(*nolint) + "\n" +
				use.If(*chainValue, "").ElseGet(sum.Of("if ", receiver, " == nil {\n", "return ", op.IfElse(*buildValue, "", "&"), buildedType, typeParams, " {}\n", "}\n")) +
				"return " + op.IfElse(*buildValue, "", "&") + buildedType + typeParams + " {\n" + parts.constructorMethodBody + "}\n" +
				"}\n"

			builderBody := util.TypeString(btyp, g.OutPkgPath) + " struct {" + generator.NoLint(*nolint) + "\n" + parts.structBody + "}"

			if !*light {
				if err := g.AddFuncOrMethod(builderConstructorMethodName, builderConstructorMethodBody); err != nil {
					return err
				}
			}
			s := generator.Structure{Name: builderName, Body: builderBody}
			if err := s.AddMethod(constrMethodName, instanceConstructorMethodBody); err != nil {
				return err
			}

			for i := range parts.fieldMethodNames {
				if err := s.AddMethod(parts.fieldMethodNames[i], parts.fieldMethodBodies[i]); err != nil {
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
				typeName := model.TypeName()
				instanceReceiver := generator.TypeReceiverVar(typeName)

				b, pre, err := generateToBuilderMethodParts(g, model, instanceReceiver, "", !(*buildValue), exportFields)
				if err != nil {
					return err
				}
				return g.AddMethod(typeName, *toBuilderMethodName, "func "+get.If(len(pkgName) > 0,
					sum.Of(*toBuilderMethodName, typeParamsDecl, "(", instanceReceiver, " ", instanceType, ") "),
				).ElseGet(
					sum.Of("(", instanceReceiver, " ", instanceType, ") ", *toBuilderMethodName, "() "),
				)+builderType+" {"+generator.NoLint(*nolint)+"\n"+get.If(!*buildValue,
					sum.Of("if ", instanceReceiver, " == nil {\nreturn ", builderInstantiate, " {}\n", "}\n"),
				).Else("")+pre+"return "+builderInstantiate+" {\n"+b+"\n"+"}\n}\n")
			}

			return nil
		},
	)
}

type builderParts struct {
	constructorMethodBody, structBody   string
	fieldMethodNames, fieldMethodBodies []string
}

func generateBuilderParts(
	g *generator.Generator, model *struc.Model, uniques map[string]string, receiverVar, typeName, setterPrefix string,
	isReceiverReference, noMethods, exportMethods, exportFields, nolint bool,
) (*builderParts, error) {
	logger.Debugf("generate builder parts: receiver %v, type %v, setterPrefix %v", receiverVar, typeName, setterPrefix)
	constructorMethodBody := ""
	structBody := ""

	fieldMethodBodies := []string{}
	fieldMethodNames := []string{}
	for i, fieldName := range model.FieldNames {
		if i > 0 {
			structBody += "\n"
		}
		if fieldType := model.FieldsType[fieldName]; fieldType.Embedded {
			if fullFieldType, err := g.GetFullFieldTypeName(fieldType, true); err != nil {
				return nil, err
			} else if embedParts, err := generateBuilderParts(g, fieldType.Model, uniques, receiverVar, typeName, setterPrefix,
				isReceiverReference, noMethods, exportMethods, exportFields, nolint); err != nil {
				return nil, err
			} else {
				init := get.If(fieldType.RefCount > 0, sum.Of("&", fullFieldType)).Else(fullFieldType)
				constructorMethodBody += fieldName + ": " + init + "{\n" + embedParts.constructorMethodBody + "\n}"
				structBody += embedParts.structBody
				if !noMethods {
					fieldMethodBodies = append(fieldMethodBodies, embedParts.fieldMethodBodies...)
					fieldMethodNames = append(fieldMethodNames, embedParts.fieldMethodNames...)
				}
			}
		} else {
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
			if err != nil {
				return nil, err
			}
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			if dupl, ok := uniques[builderField]; ok {
				return nil, fmt.Errorf("duplicated builder fields: name '%s', first type '%s', second '%s'", builderField, dupl, fullFieldType)
			}
			uniques[builderField] = fullFieldType
			constructorMethodBody += fieldName + ": " + receiverVar + "." + builderField
			structBody += builderField + " " + fullFieldType
			if !noMethods {
				fieldMethodName := generator.LegalIdentName(generator.IdentName(setterPrefix+builderField, exportMethods))
				arg := generator.LegalIdentName(generator.IdentName(builderField, false))

				fieldMethod := "func (" + receiverVar + " " + typeName + ") " + fieldMethodName + "(" + arg + " " + fullFieldType + ") " + typeName +
					" {" + generator.NoLint(nolint) + "\n" +
					get.If(isReceiverReference, sum.Of("if ", receiverVar, " != nil {\n")).Else("") +
					receiverVar + "." + builderField + "=" + arg + "\n" +
					op.IfElse(isReceiverReference, "}\n", "") +
					"return " + receiverVar + "\n}\n"
				fieldMethodBodies = append(fieldMethodBodies, fieldMethod)
				fieldMethodNames = append(fieldMethodNames, fieldMethodName)
			}
		}
		constructorMethodBody += ",\n"
	}
	return &builderParts{constructorMethodBody, structBody, fieldMethodNames, fieldMethodBodies}, nil
}

func generateToBuilderMethodParts(
	g *generator.Generator, model *struc.Model, receiver, fieldPrefix string, isReceiverReference, exportFields bool,
) (methodBody string, varsPart string, err error) {
	logger.Debugf("generate toBuilder method: receiver %v", receiver)
	for _, fieldName := range model.FieldNames {
		if fieldType := model.FieldsType[fieldName]; fieldType.Embedded {
			if fieldPath, conditionalPath, conditions := generator.FiledPathAndAccessCheckCondition(
				receiver, false, false, []generator.FieldInfo{{Name: fieldType.Name, Type: fieldType}},
			); len(conditions) > 0 {
				if embedMethodBodyPart, vars, initVars, err := generateToBuilderMethodConditionedParts(
					fieldType.Model, fieldPath, conditionalPath, conditions, receiver, isReceiverReference, exportFields,
				); err != nil {
					return "", "", err
				} else {
					methodBody += embedMethodBodyPart
					varsPart += get.If(len(vars) > 0, sum.Of("var (\n", strings.Join(vars, "\n"), "\n)\n")).Else("") + initVars
				}
			} else if methodBodyPart, _, err := generateToBuilderMethodParts(
				g, fieldType.Model, receiver, fieldType.Name, isReceiverReference, exportFields,
			); err != nil {
				return "", "", err
			} else {
				methodBody += methodBodyPart
			}
		} else {
			builderField := generator.LegalIdentName(generator.IdentName(fieldName, exportFields))
			methodBody += builderField + ": " + receiver + "." + get.If(len(fieldPrefix) > 0, sum.Of(fieldPrefix, ".")).Else("") + fieldName + ",\n"
		}
	}
	return methodBody, get.If(len(varsPart) > 0, sum.Of(varsPart, "\n")).Else(""), nil
}

func generateToBuilderMethodConditionedParts(
	model *struc.Model, parentPath, conditionalPath string, conditions []string, receiver string, isReceiverReference, exportFields bool,
) (string, []string, string, error) {
	initVars := ""
	variables := []string{}
	methodBody := ""

	varsConditionStart, varsConditionEnd := split.AndReduce(conditions, wrap.By("if ", " {\n"), replace.By("}\n"), op.Sum, op.Sum)

	initVars += varsConditionStart

	for _, fieldName := range model.FieldNames {
		fieldType := model.FieldsType[fieldName]
		if fieldType.Embedded {
			fieldPath, conditionalPath, subConditions := generator.FiledPathAndAccessCheckCondition(conditionalPath, false, false, []generator.FieldInfo{{Name: fieldType.Name, Type: fieldType}})
			fullFielPath := parentPath + get.If(len(fieldPath) > 0, sum.Of(".", fieldPath)).Else("")
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
			methodBody += builderField + ": " + varName + ",\n"
		}
	}

	initVars += varsConditionEnd

	return methodBody, variables, initVars, nil
}
