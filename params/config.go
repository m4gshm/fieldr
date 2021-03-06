package params

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
)

const (
	Name                = "fieldr"
	DefaultFileSuffix   = "_" + Name + ".go"
	CommentConfigPrefix = "go:" + Name
)

func NewConfig(flagSet *flag.FlagSet) *Config {
	return &Config{
		Type:           flagSet.String("type", "", "type name; must be set"),
		BuildTags:      multiVal(flagSet, "buildTag", []string{"fieldr", "fieldr_const_template"}, "include build tag"),
		Output:         flagSet.String("out", "", "output file name; default srcdir/<type>"+DefaultFileSuffix),
		Input:          inFlag(flagSet),
		PackagePattern: flagSet.String("package", ".", "used package"),
		Generator:      newGeneratorConfig(flagSet),
		Content:        newGeneratorContentConfig(flagSet),
	}
}

func inFlag(flagSet *flag.FlagSet) *[]string {
	return multiVal(flagSet, "in", []string{}, "go source file")
}

func newGeneratorConfig(flagSet *flag.FlagSet) *generator.Config {
	return &generator.Config{
		IncludeFieldTags: flagSet.String("filedTags", "", "comma-separated list of used field tags"),
		Nolint:           flagSet.Bool("nolint", false, "add //nolint comment"),
		OutBuildTags:     flagSet.String("outBuildTag", "", "add build tag to generated file"),
		OutPackage:       flagSet.String("outPackage", "", "output package name"),
		WrapType:         flagSet.Bool("wrap", false, "wrap tag const by own type"),
		HardcodeValues:   flagSet.Bool("hardcode", false, "hardcode tag values into generated variables, methods"),
		Name:             flagSet.String("name", "", "rename generated function to defined name"),
		ExcludeFields:    multiVal(flagSet, "excludeFields", []string{}, "exclude values from generated function result for defined fields"),
		FieldValueRewriters: multiVal(flagSet, "rewrite", []string{}, "field value rewriting applied to generated functions; "+
			"format - "+transformFieldValueFormat),
		ReturnRefs:  flagSet.Bool("ref", false, "return field as refs in generated methods"),
		Export:      flagSet.Bool("export", false, "export generated types, constant, methods"),
		NoReceiver:  flagSet.Bool("noReceiver", false, "generate no receiver-based methods for structure type"),
		ExportVars:  flagSet.Bool("exportVars", false, "export generated variables only"),
		AllFields:   flagSet.Bool("allFields", false, "include all fields (not only exported) in generated content"),
		NoEmptyTag:  flagSet.Bool("noEmptyTag", false, "exclude tags without value"),
		Snake:       flagSet.Bool("snake", false, "use snake case for constants, vars"),
		Flat:        multiVal(flagSet, "flat", []string{}, "apply generator to fields of nested structs. Used byAsMap, const and etc"),
		Compact:     flagSet.Bool("compact", false, "generate compact (in one line) array expressions"),
		ConstLength: flagSet.Int("constLen", generator.DefaultConstLength, "max cons length in line"),
		ConstReplace: multiVal(flagSet, "constReplace", []string{}, "constant's part (ident) replacers; "+
			"format - "+constReplacersFormat),
	}
}

const constReplacersFormat = "replaced_ident" + struc.ReplaceableValueSeparator + "replacer_ident" + struc.ListValuesSeparator + "replaced_ident2" + struc.ReplaceableValueSeparator + "replacer_ident"

const transformerTriggers = "<no condition (empty)>, " + string(generator.RewriteTriggerType) + ", " + string(generator.RewriteTriggerField)

var transformFieldValueFormat = "trigger" + struc.KeyValueSeparator + "trigger_value" + struc.KeyValueSeparator + "engine" +
	struc.ReplaceableValueSeparator + "engine_format" + "; supported triggers '" + transformerTriggers +
	"', engine '" + string(generator.RewriteEngineFmt) + "'"

func newGeneratorContentConfig(flagSet *flag.FlagSet) *generator.ContentConfig {
	return &generator.ContentConfig{
		Constants: multiVal(flagSet, "const", []string{}, "generate constant based on template constant; "+
			"format - consName"+struc.KeyValueSeparator+"constTemplateName"+struc.KeyValueSeparator+constReplacersFormat),
		EnumFields:       flagSet.Bool("EnumFields", false, "force to generate field constants"),
		EnumTags:         flagSet.Bool("EnumTags", false, "force to generate tag constants"),
		EnumTagValues:    flagSet.Bool("EnumTagValues", false, "force to generate tag value constants"),
		Fields:           flagSet.Bool("Fields", false, "generate Fields list var"),
		Tags:             flagSet.Bool("Tags", false, "generate Tags list var"),
		FieldTagsMap:     flagSet.Bool("FieldTagsMap", false, "generate FieldTags map var"),
		TagValuesMap:     flagSet.Bool("TagValuesMap", false, "generate TagValues map var"),
		TagValues:        multiVal(flagSet, "TagValues", []string{}, "generate TagValues var per tag"),
		TagFieldsMap:     flagSet.Bool("TagFieldsMap", false, "generate TagFields map var"),
		FieldTagValueMap: flagSet.Bool("FieldTagValueMap", false, "generate FieldTagValue map var"),

		GetFieldValue:              flagSet.Bool("GetFieldValue", false, "generate GetFieldValue func"),
		GetFieldValueByTagValue:    flagSet.Bool("GetFieldValueByTagValue", false, "generate GetFieldValueByTagValue func"),
		GetFieldValuesByTagGeneric: flagSet.Bool("GetFieldValuesByTag_", false, "generate generic GetFieldValuesByTag func with tagName argument"),
		GetFieldValuesByTag:        multiVal(flagSet, "GetFieldValuesByTag", []string{}, "generate GetFieldValuesByTag<TAG_NAME> func"),
		AsMap:                      flagSet.Bool("AsMap", false, "generate AsMap func"),
		AsTagMap:                   flagSet.Bool("AsTagMap", false, "generate AsTagMap func"),

		Strings:  flagSet.Bool("Strings", false, "generate Strings func for list types (field, tag, tag values)"),
		Excludes: flagSet.Bool("Excludes", false, "generate Excludes func for list types (field, tag, tag values)"),
	}
}

type Config struct {
	Type           *string
	BuildTags      *[]string
	Output         *string
	Input          *[]string
	PackagePattern *string

	Generator *generator.Config
	Content   *generator.ContentConfig
}

func (c *Config) MergeWith(src *Config, constantReplacers map[string]string) (config *Config, err error) {
	logger.Debugw("config merging", "dest", c, "src", src)

	if src == nil {
		return c, nil
	}
	if len(*c.Type) == 0 {
		c.Type = src.Type
	}
	inputBuildTags := *c.BuildTags
	if len(inputBuildTags) == 0 {
		c.BuildTags = src.BuildTags
	}
	if len(*c.Output) == 0 {
		c.Output = src.Output
	}
	input := *c.Input
	srcInput := *src.Input
	if len(input) > 0 && len(srcInput) > 0 {
		input = append(input, srcInput...)
		c.Input = &input
	}

	c.Generator, err = c.Generator.MergeWith(src.Generator, constantReplacers)
	if err != nil {
		return nil, err
	}
	logger.Debugw("config merged", "dest", c)
	return c, nil
}
