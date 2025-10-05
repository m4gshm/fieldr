package command

import (
	"flag"

	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/immutable"
	"github.com/m4gshm/gollections/collection/mutable"
	"github.com/m4gshm/gollections/op"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/generator/constructor"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"
)

func NewNewOpt() *Command {
	const (
		cmdName = "new-opt"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		suffix          = flagSet.String("suffix", "With", "option function suffix, use "+generator.Autoname+" for autoname <Type name>FieldName")
		name            = flagSet.String("name", generator.Autoname, "constructor name, use "+generator.Autoname+" for autoname New<Type name>")
		noConstructor   = flagSet.Bool("options-only", false, "generate option functions only")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		returnVal       = flagSet.Bool("return-value", false, "returns value instead of pointer")
		flat            = flagSet.Bool("flat", false, "makes fields of emmbedded types constructor arguments")
		nolint          = params.Nolint(flagSet)
		required        = params.MultiValFixed(flagSet, "required", nil, nil, "required arguments")
	)

	return New(
		cmdName, "generates a struct creation function with optional arguments",
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
			requird := immutable.NewSet(*required...)
			if !(*noConstructor) {
				params := typeparams.New(model.Typ.TypeParams())
				typeParams, typeParamsDecl := params.IdentDeclStrings(g.OutPkgPath)
				uniqueNames := unique.NewNamesWith(unique.DistinctBySuffix("_"))
				params.Names(g.OutPkgPath).ForEach(uniqueNames.Add)

				args, createInstance, err := constructor.GenerateConstructorArgs(g, uniqueNames, "", model.TypeName(),
					typeParams, model.FieldsNameAndType, *returnVal, *flat, requird.Contains)
				if err != nil {
					return err
				}
				arguments := args + "opts... func(*" + model.TypeName() + typeParams + ")" + op.IfElse(len(args) > 0, ",\n", "")
				constrName, constructorBody := constructor.New(*name, model.TypeName(), typeParamsDecl,
					typeParams, uniqueNames.Get("r"), *returnVal, !(*noExportMethods), *nolint,
					arguments, createInstance,
					func(receiver string) string {
						return "for _, opt := range opts {\nopt(" + op.IfElse(*returnVal, "&", "") + receiver + ")\n}"
					})
				if err := g.AddFuncOrMethod(constrName, constructorBody); err != nil {
					return err
				}
			}
			rec := generator.TypeReceiverVar(model.TypeName())
			if suffix != nil && *suffix == generator.Autoname {
				*suffix = model.TypeName()
			}
			fieldMethods, err := generateOptionFuncs(g, model, model, pkgName, rec, *suffix, !(*noExportMethods), *nolint, nil, requird.Contains)
			if err != nil {
				return err
			}
			for fieldMethodName, fieldMethodBody := range fieldMethods.All {
				if err := g.AddFuncOrMethod(fieldMethodName, fieldMethodBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func generateOptionFuncs(
	g *generator.Generator, baseModel, fieldsModel *struc.Model, pkgName, receiverVar, suffix string,
	exportMethods, nolint bool, parentFieldInfo []generator.FieldInfo, isExclude func(struc.FieldName) bool,
) (collection.Map[string, string], error) {
	logger.Debugf("generate option function: receiver %s, type %s, suffix %s", receiverVar, baseModel.TypeName(), suffix)
	fieldMethods := mutable.NewMapOrdered[string, string]()
	for fieldName, fieldType := range fieldsModel.FieldsNameAndType {
		if !isAccessible("option function", pkgName, fieldName, fieldType) {
			continue
		} else if fieldType.Embedded {
			embeddedFieldMethods, err := generateOptionFuncs(
				g, baseModel, fieldType.Model, pkgName, receiverVar, suffix, exportMethods, nolint,
				append(parentFieldInfo, generator.FieldInfo{Name: fieldType.Name, Type: fieldType}), isExclude)
			if err != nil {
				return nil, err
			}
			fieldMethods.SetMap(embeddedFieldMethods)
		} else if !isExclude(fieldName) {
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
			if err != nil {
				return nil, err
			}
			funcName := generator.IdentName(suffix+generator.LegalIdentName(generator.IdentName(fieldName, true)), exportMethods)
			logger.Debugf("option function name: %s", funcName)
			funcBody := generator.GenerateOptionFieldFunc(baseModel, pkgName, receiverVar, funcName, fieldName, fullFieldType,
				g.OutPkgPath, nolint, parentFieldInfo)
			fieldMethods.Set(funcName, funcBody)
		}
	}
	return fieldMethods, nil
}
