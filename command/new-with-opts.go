package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/mutable"
)

func NewConstructWithOptions() *Command {
	const (
		cmdName = "construct-with-options"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		suffix          = flagSet.String("option-suffix", "With", "option function suffix")
		constructorName = flagSet.String("name", generator.Autoname, "constructor function name, use "+generator.Autoname+" for autoname (New<Type name> as default)")
		noConstructor   = flagSet.Bool("no-constructor", false, "generate options only")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		useTypePrefix   = flagSet.Bool("type-prefix", false, "use type name as option function prefix")
		nolint          = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates a structure constructor with options",
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

			rec := generator.TypeReceiverVar(model.TypeName())
			if !(*noConstructor) {
				typeParams := TypeParamsString(model, g)
				typeParamsDecl := TypeParamsDeclarationString(model, g)
				constrName, constructorBody := GenerateConstructor(*constructorName, model.TypeName(), typeParamsDecl, typeParams, !(*noExportMethods), *nolint,
					"opts... func(*"+model.TypeName()+typeParams+")",
					func(receiver string) string { return "for _, opt := range opts {\nopt(" + receiver + ")\n}" })
				g.AddFuncOrMethod(constrName, constructorBody)
			}
			fieldMethods, err := generateOptionFuncs(g, model, model, pkgName, rec, *suffix, *useTypePrefix, !(*noExportMethods), *nolint, nil)
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
	useTypePrefix, exportMethods, nolint bool, parentFieldInfo []generator.FieldInfo,
) (collection.Map[string, string], error) {
	logger.Debugf("generate option function: receiver %s, type %s, suffix %s", receiverVar, baseModel.TypeName(), suffix)
	fieldMethods := mutable.NewMapOrdered[string, string]()
	for fieldName, fieldType := range fieldsModel.FieldsNameAndType {
		if !isAccessible("option function", pkgName, fieldName, fieldType) {
			continue
		} else if fieldType.Embedded {
			embeddedFieldMethods, err := generateOptionFuncs(
				g, baseModel, fieldType.Model, pkgName, receiverVar, suffix, useTypePrefix, exportMethods, nolint,
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
