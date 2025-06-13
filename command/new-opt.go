package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/generator/constructor"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"

	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/mutable"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/seq"
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
		nolint          = params.Nolint(flagSet)
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
			if !(*noConstructor) {
				params := typeparams.New(model.Typ.TypeParams())
				typeParams := params.IdentString(g.OutPkgPath)
				typeParamsDecl := params.DeclarationString(g.OutPkgPath)
				uniqueNames := unique.NewNamesWith(unique.DistinctBySuffix("_"))
				seq.ForEach(params.Names(g.OutPkgPath), uniqueNames.Add)

				constrName, constructorBody := constructor.New(*name, model.TypeName(), typeParamsDecl,
					typeParams, uniqueNames.Get("r"), *returnVal, !(*noExportMethods), *nolint,
					"opts... func(*"+model.TypeName()+typeParams+")", "",
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
			fieldMethods, err := generateOptionFuncs(g, model, model, pkgName, rec, *suffix, !(*noExportMethods), *nolint, nil)
			if err != nil {
				return err
			}
			for fieldMethodName, fieldMethodBody := range fieldMethods.All {
				if err := g.AddMethod(model.TypeName(), fieldMethodName, fieldMethodBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func generateOptionFuncs(
	g *generator.Generator, baseModel, fieldsModel *struc.Model, pkgName, receiverVar, suffix string,
	exportMethods, nolint bool, parentFieldInfo []generator.FieldInfo,
) (collection.Map[string, string], error) {
	logger.Debugf("generate option function: receiver %s, type %s, suffix %s", receiverVar, baseModel.TypeName(), suffix)
	fieldMethods := mutable.NewMapOrdered[string, string]()
	for fieldName, fieldType := range fieldsModel.FieldsNameAndType {
		if !isAccessible("option function", pkgName, fieldName, fieldType) {
			continue
		} else if fieldType.Embedded {
			embeddedFieldMethods, err := generateOptionFuncs(
				g, baseModel, fieldType.Model, pkgName, receiverVar, suffix, exportMethods, nolint,
				append(parentFieldInfo, generator.FieldInfo{Name: fieldType.Name, Type: fieldType}))
			if err != nil {
				return nil, err
			}
			fieldMethods.SetMap(embeddedFieldMethods)
		} else {
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
