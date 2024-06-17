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

func NewStringifyEnum() *Command {
	const (
		name = "enrich-enum"
	)
	type apiMethod string
	const (
		to_string  apiMethod = "name"
		all        apiMethod = "all"
		from_name  apiMethod = "from-name"
		from_value apiMethod = "from-value"
		from_index apiMethod = "from-index"
	)
	var (
		flagSet             = flag.NewFlagSet(name, flag.ExitOnError)
		toStringMethodName  = flagSet.String("name-method", "Name", "generate method that returns name of a constant")
		fromNameMethodName  = flagSet.String("from-name-func", generator.Autoname, "TODO, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixByName+" as default)")
		fromValueMethodName = flagSet.String("from-value-func", generator.Autoname, "TODO, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixByValue+" as default)")
		valuesMethodName    = flagSet.String("all-func", generator.Autoname, "all constants function name, use "+generator.Autoname+" for autoname (<Type name>"+generator.DefaultMethodSuffixAll+" as default)")
		export              = params.Export(flagSet)
		nolint              = params.Nolint(flagSet)
	)
	defaultApis := slice.Of(to_string, from_name, from_value, all)
	allowedApis := slice.Of(to_string, from_name, from_value, all)
	apis, err := flagenum.Multiple(flagSet, "api", defaultApis, allowedApis, fromString[apiMethod], toString[apiMethod], "generated api method or functions")
	if err != nil {
		panic(err)
	}

	selectedApis := immutable.NewSet(*apis...)
	return New(
		name, "enriches an enum type with a convert to string method, a convert string to the enum value funxtion and a values enumeration function",
		flagSet,
		func(context *Context) error {
			g := context.Generator
			model, err := context.EnumModel()
			if err != nil {
				return err
			}
			constValNamesMap := ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
			typ := model.Typ()
			if selectedApis.Contains(to_string) {
				funcName, funcBody, err := g.GenerateEnumName(typ, constValNamesMap, *toStringMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(all) {
				funcName, funcBody, err := g.GenerateEnumValues(typ, constValNamesMap, *valuesMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(from_name) {
				funcName, funcBody, err := g.GenerateEnumFromName(typ, constValNamesMap.Values(), *fromNameMethodName, *export, *nolint)
				if err != nil {
					return err
				} else if err = g.AddFuncOrMethod(funcName, funcBody); err != nil {
					return err
				}
			}
			if selectedApis.Contains(from_value) {
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
