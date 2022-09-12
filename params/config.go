package params

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
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
		BuildTags:      MultiVal(flagSet, "buildTag", []string{"fieldr", "fieldr_const_template"}, "include build tag"),
		Output:         flagSet.String("out", "", "output file name; default srcdir/<type>"+DefaultFileSuffix),
		Input:          inFlag(flagSet),
		PackagePattern: flagSet.String("package", ".", "used package"),
		OutBuildTags:   flagSet.String("out-build-tag", "", "add build tag to generated file"),
		OutPackage:     flagSet.String("out-package", "", "output package name"),
	}
}

func inFlag(flagSet *flag.FlagSet) *[]string {
	return MultiVal(flagSet, "in", []string{}, "go source file")
}

type GentConfig struct {
	Nolint              *bool
	Export              *bool
	NoReceiver          *bool
	ExportVars          *bool
	AllFields           *bool
	ReturnRefs          *bool
	WrapType            *bool
	HardcodeValues      *bool
	NoEmptyTag          *bool
	Compact             *bool
	Snake               *bool
	Flat                *[]string
	ConstLength         *int
	ConstReplace        *[]string
	Name                *string
	ExcludeFields       *[]string
	FieldValueRewriters *[]string
}

func WithPrivate(flagSet *flag.FlagSet) *bool {
	return flagSet.Bool("with-private", false, "use private fields for generating content")
}

func Snake(flagSet *flag.FlagSet) *bool {
	return flagSet.Bool("snake", false, "use snake case in generated content naming")
}

func Nolint(flagSet *flag.FlagSet) *bool {
	return flagSet.Bool("nolint", false, "add 'nolint' comment to generated content")
}

func ExportCont(flagSet *flag.FlagSet, content string) *bool {
	return flagSet.Bool("export", false, "export generated "+content)
}

func Export(flagSet *flag.FlagSet) *bool {
	return ExportCont(flagSet, "content")
}

const constReplacersFormat = "replaced_ident" + struc.ReplaceableValueSeparator + "replacer_ident" + struc.ListValuesSeparator + "replaced_ident2" + struc.ReplaceableValueSeparator + "replacer_ident"

const transformerTriggers = "<no condition (empty)>, " + string(generator.RewriteTriggerType) + ", " + string(generator.RewriteTriggerField)

var transformFieldValueFormat = "trigger" + struc.KeyValueSeparator + "trigger_value" + struc.KeyValueSeparator + "engine" +
	struc.ReplaceableValueSeparator + "engine_format" + "; supported triggers '" + transformerTriggers +
	"', engine '" + string(generator.RewriteEngineFmt) + "'"

const enum_field_const = "enum-const"

func newGeneratorContentConfig(flagSet *flag.FlagSet) *generator.ContentConfig {
	return &generator.ContentConfig{
		Constants: MultiVal(flagSet, "const", []string{}, "generate constant based on template constant; "+
			"format - consName"+struc.KeyValueSeparator+"constTemplateName"+struc.KeyValueSeparator+constReplacersFormat),
	}
}

type Config struct {
	Type           *string
	BuildTags      *[]string
	Output         *string
	Input          *[]string
	PackagePattern *string
	OutBuildTags   *string
	OutPackage     *string
}
