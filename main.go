package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/struc"

	"golang.org/x/tools/go/packages"
)

const name = "fieldr"
const _type = "type"
const defaultSuffix = "_" + name + ".go"

const defBuildTag = "fieldr_const_template"

var (
	TagParsers    = struc.TagValueParsers{}
	ExcludeValues = map[struc.TagName]map[struc.TagValue]bool{}
)

var (
	typ            = flag.String(_type, "", "type name; must be set")
	inputBuildTags = multiflag("inTag", []string{defBuildTag}, "input build tag")
	outBuildTags   = flag.String("outTag", "", "add build tag to generated file")
	output         = flag.String("out", "", "output file name; default srcdir/<type>"+defaultSuffix)
	input          = multiflag("in", []string{}, "go source file")
	tag            = flag.String("tag", "", "tag used to constant naming")
	wrap           = flag.Bool("wrap", false, "wrap tag const by own type")
	hardcode       = flag.Bool("hardcode", false, "hardcode tag values intogenerated variables, methods")
	ref            = flag.Bool("ref", false, "return field as refs in generated methods")
	export         = flag.Bool("export", false, "export generated types, constant, methods")
	exportVars     = flag.Bool("exportVars", false, "export generated variables only")
	allFields      = flag.Bool("allFields", false, "include all fields (not only exported) in generated content")
	noEmptyTag     = flag.Bool("noEmptyTag", false, "exclude tags without value")
	compact        = flag.Bool("compact", false, "generate compact (in one line) array expressions")
	constants      = multiflag("const", []string{}, "templated constant for generating field's tag based constant; format - consName:constTemplateName:replaced_ident=replacer_ident,replaced_ident2=replacer_ident")
	constLength    = flag.Int("constLen", 80, "max cons length in line")
	constReplace   = flag.String("constReplace", "", "constant's part (ident) replacers; format - replaced_ident=replacer_ident,replaced_ident2=replacer_ident")
	packagePattern = flag.String("package", ".", "used package")

	generateContentOptions = generator.GenerateContentOptions{
		EnumFields:       flag.Bool("EnumFields", false, "force to generate field constants"),
		EnumTags:         flag.Bool("EnumTags", false, "force to generate tag constants"),
		EnumTagValues:    flag.Bool("EnumTagValues", false, "force to generate tag value constants"),
		Fields:           flag.Bool("Fields", false, "generate Fields list var"),
		Tags:             flag.Bool("Tags", false, "generate Tags list var"),
		FieldTagsMap:     flag.Bool("FieldTagsMap", false, "generate FieldTags map var"),
		TagValuesMap:     flag.Bool("TagValuesMap", false, "generate TagValues map var"),
		TagValues:        multiflag("TagValues", []string{}, "generate TagValues var per tag"),
		TagFieldsMap:     flag.Bool("TagFieldsMap", false, "generate TagFields map var"),
		FieldTagValueMap: flag.Bool("FieldTagValueMap", false, "generate FieldTagValue map var"),

		GetFieldValue:              flag.Bool("GetFieldValue", false, "generate GetFieldValue func"),
		GetFieldValueByTagValue:    flag.Bool("GetFieldValueByTagValue", false, "generate GetFieldValueByTagValue func"),
		GetFieldValuesByTagGeneric: flag.Bool("GetFieldValuesByTag_", false, "generate GetFieldValuesByTag func with tagName argument"),
		GetFieldValuesByTag:        multiflag("GetFieldValuesByTag", []string{}, "generate GetFieldValuesByTag<TAG_NAME> func, omit tag name to generate generic function"),
		AsMap:                      flag.Bool("AsMap", false, "generate AsMap func"),
		AsTagMap:                   flag.Bool("AsTagMap", false, "generate AsTagMap func"),

		Strings:  flag.Bool("Strings", false, "generate Strings func for list types (field, tag, tag values)"),
		Excludes: flag.Bool("Excludes", false, "generate Excludes func for list types (field, tag, tag values)"),
	}
)

type Multiflag struct {
	values []string
}

func (f *Multiflag) String() string {
	return strings.Join(f.values, ",")
}

func (f *Multiflag) Set(s string) error {
	if f.values == nil {
		f.values = []string{}
	}
	f.values = append(f.values, s)
	return nil
}

func (f *Multiflag) Get() interface{} { return f.values }

func multiflag(name string, defValues []string, usage string) *[]string {
	values := Multiflag{values: defValues}
	flag.Var(&values, name, usage)
	return &values.values
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of "+name+":\n")
	fmt.Fprintf(os.Stderr, "\t"+name+" [flags] -type T [directory]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetPrefix(name + ": ")
	flag.Usage = usage
	flag.Parse()

	typeName := *typ
	if len(typeName) == 0 {
		log.Print("no type arg")
		flag.Usage()
		os.Exit(2)
	}

	args := flag.Args()
	outputDir := outDir(args)
	if len(outputDir) > 0 {
		if err := os.Chdir(outputDir); err != nil {
			log.Fatalf("out dir error: %v", err)
		}
	}

	pkg := extractPackage(*inputBuildTags, *packagePattern)
	packageName := pkg.Name
	files := pkg.Syntax
	if len(files) == 0 {
		log.Printf("no src files in package %s", packageName)
		return
	}

	fileSet := token.NewFileSet()

	for _, srcFile := range *input {
		file, err := parser.ParseFile(fileSet, srcFile, nil, 0)
		if err != nil {
			log.Fatal(err)
		}
		files = append(files, file)

	}
	typeFile, err := struc.FindStructTags(files, typeName, struc.TagName(*tag), TagParsers, ExcludeValues, *constants, *constReplace)
	if err != nil {
		log.Fatal(err)
	}
	if typeFile == nil {
		log.Printf("type not found, %s", typeName)
		return
	}

	var (
		generateAll  = true
		optionFields = reflect.ValueOf(generateContentOptions)
		field        = optionFields.NumField()
	)
	for i := 0; i < field; i++ {
		structField := optionFields.Field(i)
		sfk := structField.Kind()
		if sfk == reflect.Ptr {
			elem := structField.Elem()
			noGenerate := isNoGenerate(elem)
			generateAll = generateAll && noGenerate
			if !generateAll {
				break
			}
		}
	}

	if generateAll {
		generateAll = len(*constants) == 0
	}

	generateContentOptions.All = generateAll

	onlyExported := !*allFields
	//g := generator.NewGenerator(name, *wrap, *hardcode, *ref, *export, onlyExported, *exportVars, *compact, *noEmptyTag, *constants, *constLength, &generateContentOptions)
	g := generator.Generator{
		Name:           name,
		WrapType:       *wrap,
		HardcodeValues: *hardcode,
		ReturnRefs:     *ref,
		Export:         *export,
		OnlyExported:   onlyExported,
		ExportVars:     *exportVars,
		Compact:        *compact,
		NoEmptyTag:     *noEmptyTag,
		Constants:      *constants,
		ConstLength:    *constLength,
		Opts:           &generateContentOptions,
		OutBuildTags:   *outBuildTags,
	}

	err = g.GenerateFile(typeFile)
	if err != nil {
		log.Fatalf("generate file error: %s", err)
	}
	src, fmtErr := g.FormatSrc()

	outputName := *output
	if outputName == "" {
		baseName := typeName + defaultSuffix
		outputName = strings.ToLower(baseName)
	}
	const userWriteOtherRead = fs.FileMode(0644)
	if writeErr := ioutil.WriteFile(outputName, src, userWriteOtherRead); writeErr != nil {
		log.Fatalf("writing output: %s", writeErr)
	} else if fmtErr != nil {
		log.Fatalf("go src code formatting error: %s", fmtErr)
	}

}

func isNoGenerate(elem reflect.Value) bool {
	var notGenerate bool
	kind := elem.Kind()
	switch kind {
	case reflect.Bool:
		notGenerate = !elem.Bool()
	case reflect.String:
		s := elem.String()
		notGenerate = len(s) == 0
	case reflect.Slice:
		notGenerate = true
		l := elem.Len()
		for i := 0; i < l; i++ {
			value := elem.Index(i)
			ng := isNoGenerate(value)
			notGenerate = notGenerate && ng
		}
	}
	return notGenerate
}

func outDir(args []string) string {
	if len(args) > 0 && isDir(args[0]) {
		return args[0]
	}
	return ""
}

func isDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	dir := info.IsDir()
	return dir
}

func extractPackage(buildTags []string, patterns ...string) *packages.Package {
	packages, err := packages.Load(&packages.Config{
		Mode:       packages.NeedSyntax,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(buildTags, " "))},
	}, patterns...)
	if err != nil {
		log.Fatal(err)
	}
	if len(packages) != 1 {
		log.Fatalf("error: %d packages found", len(packages))
	}

	pack := packages[0]

	errors := pack.Errors
	if len(errors) > 0 {
		log.Fatal(errors[0])
	}

	return pack
}
