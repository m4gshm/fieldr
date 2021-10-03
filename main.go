package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	buildTags      = multiflag("buildTag", []string{defBuildTag}, "build tag")
	output         = flag.String("output", "", "output file name; default srcdir/<type>"+defaultSuffix)
	tag            = flag.String("tag", "", "tag used to constant naming")
	wrap           = flag.Bool("wrap", false, "wrap tag const by own type")
	ref            = flag.Bool("ref", false, "return field as refs in generated methods")
	export         = flag.Bool("export", false, "export generated types, constant, methods")
	exportVars     = flag.Bool("exportVars", false, "export generated variables only")
	allFields      = flag.Bool("allFields", false, "include all fields (not only exported) in generated content")
	noEmptyTag     = flag.Bool("noEmptyTag", false, "exclude tags without value")
	constants      = multiflag("const", []string{}, "templated constant for generating field's tag based constant")
	packagePattern = flag.String("package", ".", "used package")
	srcFiles       = multiflag("src", []string{}, "go source file")

	generateContentOptions = generator.GenerateContentOptions{
		Fields:           flag.Bool("Fields", false, "generate Fields list var"),
		Tags:             flag.Bool("Tags", false, "generate Tags list var"),
		FieldTagsMap:     flag.Bool("FieldTagsMap", false, "generate FieldTags map var"),
		TagValuesMap:     flag.Bool("TagValuesMap", false, "generate TagValues map var"),
		TagFieldsMap:     flag.Bool("TagFieldsMap", false, "generate TagFields map var"),
		FieldTagValueMap: flag.Bool("FieldTagValueMap", false, "generate FieldTagValue map var"),

		GetFieldValue:           flag.Bool("GetFieldValue", false, "generate GetFieldValue func"),
		GetFieldValueByTagValue: flag.Bool("GetFieldValueByTagValue", false, "generate GetFieldValueByTagValue func"),
		GetFieldValuesByTag:     flag.Bool("GetFieldValuesByTag", false, "generate GetFieldValuesByTag func"),
		AsMap:                   flag.Bool("AsMap", false, "generate AsMap func"),
		AsTagMap:                flag.Bool("AsTagMap", false, "generate AsTagMap func"),

		Strings: flag.Bool("Strings", false, "generate Strings func for list types (field, tag, tag values)"),
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
	values := Multiflag{}
	if defValues != nil {
		values.values = defValues
	}
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
	err := os.Chdir(outDir(args))
	if err != nil {
		log.Fatal(err)
	}

	pkg := extractPackage(*buildTags, *packagePattern)
	packageName := pkg.Name
	files := pkg.Syntax
	if len(files) == 0 {
		log.Printf("no src files in package %s", packageName)
		return
	}

	fileSet := token.NewFileSet()

	for _, srcFile := range *srcFiles {
		file, err := parser.ParseFile(fileSet, srcFile, nil, 0)
		if err != nil {
			log.Fatal(err)
		}
		files = append(files, file)

	}
	typeFile, err := findTypeFile(files, typeName, *tag, *constants)
	if err != nil {
		log.Fatal(err)
	}
	if typeFile == nil {
		log.Printf("type not found, %s", typeName)
		return
	}

	generateAll := true
	optionFields := reflect.ValueOf(generateContentOptions)
	field := optionFields.NumField()
	for i := 0; i < field; i++ {
		structField := optionFields.Field(i)
		elem := structField.Elem()
		notGenerate := elem.Kind() == reflect.Bool && !elem.Bool()
		generateAll = generateAll && notGenerate
	}

	if generateAll {
		for i := 0; i < field; i++ {
			optionFields.Field(i).Elem().SetBool(true)
		}
	}

	g := generator.NewGenerator(name, *wrap, *ref, *export, !*allFields, *exportVars, *noEmptyTag, *constants, &generateContentOptions)

	err = g.GenerateFile(typeFile)
	if err != nil {
		log.Fatalf("generate file error: %s", err)
	}
	src, fmtErr := g.FormatSrc()

	outputName := *output
	if outputName == "" {

		baseName := typeName + defaultSuffix
		outputName = filepath.Join(outDir(args), strings.ToLower(baseName))
	}
	const userWriteOtherRead = fs.FileMode(0644)
	if writeErr := ioutil.WriteFile(outputName, src, userWriteOtherRead); writeErr != nil {
		log.Fatalf("writing output: %s", writeErr)
	} else if fmtErr != nil {
		log.Fatalf("go src code formatting error: %s", fmtErr)
	}

}

func outDir(args []string) string {
	if len(args) > 0 && isDir(args[0]) {
		return args[0]
	}
	return "."
}

func isDir(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
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

func findTypeFile(files []*ast.File, typeName string, tag string, constants []string) (*struc.Struct, error) {
	return struc.FindStructTags(files, typeName, struc.TagName(tag), TagParsers, ExcludeValues, constants)
}
