package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/params"
)

func NewWith() *Command {
	const (
		cmdName = "with"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ExitOnError)
		prefix          = flagSet.String("prefix", "With", "optional methods prefix")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		noRefReceiver   = flagSet.Bool("no-ref", false, "use value type (not pointer) for methods receiver")
		nolint          = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates TODO",
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
			fmn, fmb, err := generateOptionFuncs(g, model, model, pkgName, rec, *prefix, !(*noRefReceiver), !(*noExportMethods), *nolint, nil)
			if err != nil {
				return err
			}

			for i := range fmn {
				fieldMethodName := fmn[i]
				fieldMethodBody := fmb[i]
				if err := g.AddMethod(model.TypeName(), fieldMethodName, fieldMethodBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func generateOptionFuncs(
	g *generator.Generator, baseModel, fieldsModel *struc.Model, pkgName, receiverVar, prefix string,
	isReceiverReference, exportMethods, nolint bool, parentFieldInfo []generator.FieldInfo,
) ([]string, []string, error) {
	logger.Debugf("generate with: receiver %s, type %s, prefix %s", receiverVar, baseModel.TypeName(), prefix)
	fieldMethodBodies := []string{}
	fieldMethodNames := []string{}
	for _, fieldName := range fieldsModel.FieldNames {
		fieldType := fieldsModel.FieldsType[fieldName]
		if len(pkgName) > 0 {
			if !generator.IsExported(fieldName) {
				logger.Debugf("cannot generate Opt for private field %s for package %s", fieldName, pkgName)
				continue
			}

			if m := fieldType.Model; m != nil {
				if !generator.IsExported(m.TypeName()) {
					logger.Debugf("cannot generate Opt for field %s with private type % for package %s", fieldName, m.TypeName(), pkgName)
					continue
				}
			}
		}
		if fieldType.Embedded {
			ebmeddedFieldMethodNames, ebmeddedFieldMethodBodies, err := generateOptionFuncs(
				g, baseModel, fieldType.Model, pkgName, receiverVar, prefix, isReceiverReference, exportMethods, nolint,
				append(parentFieldInfo, generator.FieldInfo{Name: fieldType.Name, Type: fieldType}))
			if err != nil {
				return nil, nil, err
			}
			fieldMethodBodies = append(fieldMethodBodies, ebmeddedFieldMethodBodies...)
			fieldMethodNames = append(fieldMethodNames, ebmeddedFieldMethodNames...)
		} else {
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
			if err != nil {
				return nil, nil, err
			}
			suffix := generator.LegalIdentName(generator.IdentName(fieldName, true))

			funcName := generator.IdentName(prefix+suffix, exportMethods)
			logger.Debugf("opt %s", funcName)
			funcBody := generator.GenerateOptionFieldFunc(baseModel, pkgName, receiverVar, funcName, fieldName, fullFieldType,
				g.OutPkgPath, nolint, isReceiverReference, parentFieldInfo)
			fieldMethodBodies = append(fieldMethodBodies, funcBody)
			fieldMethodNames = append(fieldMethodNames, funcName)
		}
	}
	return fieldMethodNames, fieldMethodBodies, nil
}
