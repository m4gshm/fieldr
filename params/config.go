package params

import (
	"flag"
)

const (
	Name                = "fieldr"
	DefaultFileSuffix   = "_" + Name + ".go"
	CommentConfigPrefix = "go:" + Name
)

func NewConfig(flagSet *flag.FlagSet) *Config {
	return &Config{
		Type:           Type(flagSet),
		BuildTags:      MultiVal(flagSet, "buildTag", []string{"fieldr"}, "include build tag"),
		Output:         flagSet.String("out", "", "output file name; default ./<type>"+DefaultFileSuffix),
		Input:          inFlag(flagSet),
		PackagePattern: flagSet.String("package", ".", "used package"),
		OutBuildTags:   flagSet.String("out-build-tag", "", "add build tag to generated file"),
		OutPackage:     flagSet.String("out-package", "", "output package name"),
	}
}

func Type(flagSet *flag.FlagSet) *string {
	return flagSet.String("type", "", "structure type used as a source for creating content")
}

func inFlag(flagSet *flag.FlagSet) *[]string {
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
	return flagSet.Bool("export", false, "export generated "+content)
}

func Export(flagSet *flag.FlagSet) *bool {
	return ExportCont(flagSet, "content")
}

func Flat(flagSet *flag.FlagSet) *[]string {
	return MultiVal(flagSet, "flat", []string{}, "apply generator to fields of nested structs")
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
