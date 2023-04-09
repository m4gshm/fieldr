package command

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

func NewGettersSetters() *Command {
	const (
		cmdName = "get-set"
	)
	var (
		flagSet         = flag.NewFlagSet(cmdName, flag.ContinueOnError)
		getterPrefix    = flagSet.String("getter-prefix", "Get", "getter methods prefix")
		setterPrefix    = flagSet.String("setter-prefix", "Set", "setter methods prefix")
		noExportMethods = flagSet.Bool("no-export", false, "no export generated methods")
		noRefReceiver   = flagSet.Bool("no-ref", false, "use value type (not pointer) for methods receiver")
		nolint          = params.Nolint(flagSet)
	)

	return New(
		cmdName, "generates getters, setters for a structure type",
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

			rec := generator.PathToShortVarName(model.TypeName)
			fmn, fmb, err := generateGettersSetters(g, model, model, pkgName, rec, *getterPrefix, *setterPrefix, !(*noRefReceiver), !(*noExportMethods), *nolint, nil)
			if err != nil {
				return err
			}

			for i := range fmn {
				fieldMethodName := fmn[i]
				fieldMethodBody := fmb[i]
				if err := g.AddMethod(model.TypeName, fieldMethodName, fieldMethodBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func generateGettersSetters(
	g *generator.Generator, baseModel, fieldsModel *struc.Model, pkgName, receiverVar, getterPrefix, setterPrefix string,
	isReceiverReference, exportMethods, nolint bool, parentFieldInfo []generator.FieldInfo,
) ([]string, []string, error) {
	logger.Debugf("generate getters, setters: receiver %s, type %s, getterPrefix %s setterPrefix %s", receiverVar, baseModel.TypeName, getterPrefix, setterPrefix)
	fieldMethodBodies := []string{}
	fieldMethodNames := []string{}
	for _, fieldName := range fieldsModel.FieldNames {
		fieldType := fieldsModel.FieldsType[fieldName]
		if fieldType.Embedded {
			ebmeddedFieldMethodNames, ebmeddedFieldMethodBodies, err := generateGettersSetters(
				g, baseModel, fieldType.Model, pkgName, receiverVar, getterPrefix, setterPrefix, isReceiverReference, exportMethods, nolint,
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
			getterName := generator.IdentName(getterPrefix+suffix, exportMethods)
			getterBody := generator.GenerateGetter(baseModel, pkgName, receiverVar, getterName, fieldName, fullFieldType, g.OutPkg.PkgPath, nolint, isReceiverReference, parentFieldInfo)
			setterName := generator.IdentName(setterPrefix+suffix, exportMethods)
			setterBody := generator.GenerateSetter(baseModel, pkgName, receiverVar, setterName, fieldName, fullFieldType, g.OutPkg.PkgPath, nolint, isReceiverReference, parentFieldInfo)
			fieldMethodBodies = append(fieldMethodBodies, getterBody, setterBody)
			fieldMethodNames = append(fieldMethodNames, getterName, setterName)
		}
	}
	return fieldMethodNames, fieldMethodBodies, nil
}
