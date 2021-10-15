package params

import (
	"flag"

	"github.com/m4gshm/fieldr/generator"
)

const (
	Name                = "fieldr"
	DefaultFileSuffix   = "_" + Name + ".go"
	DefBuildTag         = "fieldr_const_template"
	CommentConfigPrefix = "go:" + Name
)

func NewConfig(flagSet *flag.FlagSet) *Config {
	return &Config{
		Typ:            flagSet.String("type", "", "type name; must be set"),
		InputBuildTags: MultiVal(flagSet, "inTag", []string{DefBuildTag}, "input build tag"),
		Output:         flagSet.String("out", "", "output file name; default srcdir/<type>"+DefaultFileSuffix),
		Input:          InFlag(flagSet),
		Tag:            flagSet.String("tag", "", "tag used to constant naming"),
		PackagePattern: flagSet.String("package", ".", "used package"),
		Generator:      NewGeneratorConfig(flagSet),
		Content:        NewGeneratorContentConfig(flagSet),
	}
}

func InFlag(flagSet *flag.FlagSet) *[]string {
	return MultiVal(flagSet, "in", []string{}, "go source file")
}

func NewGeneratorConfig(flagSet *flag.FlagSet) *generator.Config {
	return &generator.Config{
		OutBuildTags:   flagSet.String("outTag", "", "add build tag to generated file"),
		WrapType:       flagSet.Bool("wrap", false, "wrap tag const by own type"),
		HardcodeValues: flagSet.Bool("hardcode", false, "hardcode tag values intogenerated variables, methods"),
		ReturnRefs:     flagSet.Bool("ref", false, "return field as refs in generated methods"),
		Export:         flagSet.Bool("export", false, "export generated types, constant, methods"),
		ExportVars:     flagSet.Bool("exportVars", false, "export generated variables only"),
		AllFields:      flagSet.Bool("allFields", false, "include all fields (not only exported) in generated content"),
		NoEmptyTag:     flagSet.Bool("noEmptyTag", false, "exclude tags without value"),
		Compact:        flagSet.Bool("compact", false, "generate compact (in one line) array expressions"),
		ConstLength:    flagSet.Int("constLen", generator.DefaultConstLength, "max cons length in line"),
		ConstReplace: MultiVal(flagSet, "constReplace", []string{}, "constant's part (ident) replacers; "+
			"format - replaced_ident=replacer_ident,replaced_ident2=replacer_ident"),
	}
}

func NewGeneratorContentConfig(flagSet *flag.FlagSet) *generator.ContentConfig {
	return &generator.ContentConfig{
		Constants: MultiVal(flagSet, "const", []string{}, "templated constant for generating field's tag based constant; "+
			"format - consName:constTemplateName:replaced_ident=replacer_ident,replaced_ident2=replacer_ident"),
		EnumFields:       flagSet.Bool("EnumFields", false, "force to generate field constants"),
		EnumTags:         flagSet.Bool("EnumTags", false, "force to generate tag constants"),
		EnumTagValues:    flagSet.Bool("EnumTagValues", false, "force to generate tag value constants"),
		Fields:           flagSet.Bool("Fields", false, "generate Fields list var"),
		Tags:             flagSet.Bool("Tags", false, "generate Tags list var"),
		FieldTagsMap:     flagSet.Bool("FieldTagsMap", false, "generate FieldTags map var"),
		TagValuesMap:     flagSet.Bool("TagValuesMap", false, "generate TagValues map var"),
		TagValues:        MultiVal(flagSet, "TagValues", []string{}, "generate TagValues var per tag"),
		TagFieldsMap:     flagSet.Bool("TagFieldsMap", false, "generate TagFields map var"),
		FieldTagValueMap: flagSet.Bool("FieldTagValueMap", false, "generate FieldTagValue map var"),

		GetFieldValue:              flagSet.Bool("GetFieldValue", false, "generate GetFieldValue func"),
		GetFieldValueByTagValue:    flagSet.Bool("GetFieldValueByTagValue", false, "generate GetFieldValueByTagValue func"),
		GetFieldValuesByTagGeneric: flagSet.Bool("GetFieldValuesByTag_", false, "generate GetFieldValuesByTag func with tagName argument"),
		GetFieldValuesByTag:        MultiVal(flagSet, "GetFieldValuesByTag", []string{}, "generate GetFieldValuesByTag<TAG_NAME> func, omit tag name to generate generic function"),
		AsMap:                      flagSet.Bool("AsMap", false, "generate AsMap func"),
		AsTagMap:                   flagSet.Bool("AsTagMap", false, "generate AsTagMap func"),

		Strings:  flagSet.Bool("Strings", false, "generate Strings func for list types (field, tag, tag values)"),
		Excludes: flagSet.Bool("Excludes", false, "generate Excludes func for list types (field, tag, tag values)"),
	}
}

type Config struct {
	Typ            *string
	InputBuildTags *[]string
	Output         *string
	Input          *[]string
	Tag            *string
	PackagePattern *string

	Generator *generator.Config
	Content   *generator.ContentConfig
}

func (c *Config) MergeWith(src *Config, constantReplacers map[string]string) (config *Config, err error) {
	if src == nil {
		return c, nil
	}
	if len(*c.Typ) == 0 {
		c.Typ = src.Typ
	}
	inputBuildTags := *c.InputBuildTags
	if len(inputBuildTags) == 0 {
		c.InputBuildTags = src.InputBuildTags
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

	if len(*c.Tag) == 0 {
		c.Tag = src.Tag
	}

	c.Generator, err = c.Generator.MergeWith(src.Generator, constantReplacers)
	if err != nil {
		return nil, err
	}
	return c, nil
}
