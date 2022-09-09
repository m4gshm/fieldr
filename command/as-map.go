package command

import (
	"flag"

	"github.com/m4gshm/fieldr/coderewriter"
	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
)

func NewAsMapMethod() *Command {
	const (
		name     = "as-map"
		flagVal  = "val"
		flagName = "name"
	)

	const transformerTriggers = "<no condition (empty)>, " + string(generator.RewriteTriggerType) + ", " + string(generator.RewriteTriggerField)

	var transformFieldValueFormat = "trigger" + struc.KeyValueSeparator + "trigger_value" + struc.KeyValueSeparator + "engine" +
		struc.ReplaceableValueSeparator + "engine_format" + "; supported triggers '" + transformerTriggers +
		"', engine '" + string(generator.RewriteEngineFmt) + "'"

	var (
		flagSet = flag.NewFlagSet(name, flag.ContinueOnError)

		constName           = flagSet.String("name", "", "function/method name")
		export              = params.Export(flagSet, "function/method")
		snake               = flagSet.Bool("snake", false, "use snake format for generated function/method name")
		wrap                = flagSet.Bool("wrap", false, "wrap")
		ref                 = flagSet.Bool("ref", false, "ref")
		fun                 = flagSet.Bool("func", false, "func")
		all                 = flagSet.Bool("all", false, "all")
		nolint              = flagSet.Bool("nolint", false, "nolint")
		hardcode            = flagSet.Bool("hardcode", false, "hardcode")
		fieldValueRewriters = params.MultiVal(flagSet, "rewrite", []string{}, "field value rewriting applied to generated functions; "+
			"format - "+transformFieldValueFormat)
		flat = params.MultiVal(flagSet, "flat", []string{}, "apply generator to fields of nested structs")
	)

	c := New(
		name, "generates method or functon that converts the struct type to a map",
		flagSet,
		func(g *generator.Generator, hmodel *struc.HierarchicalModel) error {
			model := toFlatModel(hmodel, *flat)
			if structPackage, err := g.StructPackage(model); err != nil {
				return err
			} else {
				fieldType := generator.GetFieldType(model.TypeName, *export, *snake)
				g.AddType(fieldType, generator.BaseConstType)
				if err := g.GenerateFieldConstants(model, fieldType, model.FieldNames, *export, *snake, *wrap); err != nil {
					return err
				} else if rewriter, err := coderewriter.New(*fieldValueRewriters); err != nil {
					return err
				} else if typeLink, funcName, funcBody, err := generator.GenerateAsMapFunc(
					g, model, structPackage, *constName, rewriter, *export, *snake, *wrap, *ref, *fun, *all, *nolint, *hardcode,
				); err != nil {
					return err
				} else if err = g.AddReceiverFunc(typeLink, funcName, funcBody, err); err != nil {
					return err
				}
				return nil
			}
		},
	)
	return c
}
