package command

import (
	"flag"
	"go/types"

	"github.com/m4gshm/flag/flagenum"
	"github.com/m4gshm/gollections/collection/immutable"
	ordermap "github.com/m4gshm/gollections/collection/immutable/ordered/map_"
	"github.com/m4gshm/gollections/slice"
	"github.com/m4gshm/gollections/slice/group"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
)

func toString[F ~string](from F) string { return string(from) }
func fromString[F ~string](s string) F  { return F(s) }

func NewEnrichConstType() *Command {
	const (
		name = "enrich-const-type"
	)
	type apiMethod string
	const (
		nameMeth      apiMethod = "getter"
		allFunc       apiMethod = "all"
		fromNameFunc  apiMethod = "from-name"
		fromValueFunc apiMethod = "from-value"
	)
	var (
		flagSet             = flag.NewFlagSet(name, flag.ExitOnError)
		toStringMethodName  = flagSet.String("get-name", "Name", "a getter name that returns the constant name")
		fromNameMethodName  = flagSet.String("from-name", generator.Autoname, "a function name that returns a constant of the set by its name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixByName+" as default)")
		fromValueMethodName = flagSet.String("from-value", generator.Autoname, "a function name that returns a constant of the set by its underlying type value, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixByValue+" as default)")
		valuesMethodName    = flagSet.String("all-func", generator.Autoname, "a function name that returns a slice contains all constants of the set, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixAll+" as default)")
		export              = params.Export(flagSet)
		nolint              = params.Nolint(flagSet)
	)
	defaultApis := slice.Of(nameMeth, fromNameFunc, fromValueFunc, allFunc)
	allowedApis := slice.Of(nameMeth, fromNameFunc, fromValueFunc, allFunc)
	apis, err := flagenum.Multiple(flagSet, "api", defaultApis, allowedApis, fromString[apiMethod], toString[apiMethod], "generated api method or functions")
	if err != nil {
		panic(err)
	}

	selectedApis := immutable.NewSet(*apis...)
	return New(
		name, "extends a constant set type with functions and methods",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			model, err := context.EnumModel()
			if err != nil {
				return err
			}
			constValNamesMap := ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
			typ := model.Typ()
			if selectedApis.Contains(nameMeth) {
				funcName, funcBody, err := g.GenerateEnumName(typ, constValNamesMap, *toStringMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(allFunc) {
				funcName, funcBody, err := g.GenerateEnumValues(typ, constValNamesMap, *valuesMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(fromNameFunc) {
				funcName, funcBody, err := g.GenerateEnumFromName(typ, constValNamesMap.Values(), *fromNameMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(fromValueFunc) {
				funcName, funcBody, err := g.GenerateEnumFromValue(typ, constValNamesMap, *fromValueMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
