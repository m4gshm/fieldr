package params

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
)

const (
	Name                = "fieldr"
	DefaultFileSuffix   = "_" + Name + ".go"
	CommentConfigPrefix = "go:" + Name
)

func NewTypeConfig(flagSet *flag.FlagSet) *TypeConfig {
	typeConfig := &TypeConfig{}
	flagSet.StringVar(&typeConfig.Type, "type", "", "structure type used as a source for creating content")
	flagSet.StringVar(&typeConfig.Output, "out", "", "output file name; default ./<type>"+DefaultFileSuffix+"; use "+generator.Autoname+" to inject generated code into the Type source file")
	flagSet.StringVar(&typeConfig.OutBuildTags, "out-build-tag", "", "add build tag to generated file")
	flagSet.StringVar(&typeConfig.OutPackage, "out-package", "", "output package name")
	return typeConfig
}

func InFlag(flagSet *flag.FlagSet) *[]string {
	return MultiVal(flagSet, "in", []string{}, "go source file")
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
	return flagSet.Bool("export", false, "no export generated "+content)
}

func Export(flagSet *flag.FlagSet) *bool {
	return ExportCont(flagSet, "content")
}

func Flat(flagSet *flag.FlagSet) *[]string {
	return MultiVal(flagSet, "flat", []string{}, "apply generator to fields of nested structs")
}

type TypeConfig struct {
	Type         string
	Output       string
	OutBuildTags string
	OutPackage   string
}
