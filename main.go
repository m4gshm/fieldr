package main

import (
	"flag"
	"fmt"
	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/struc"
	"go/ast"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const name = "fieldr"
const defaultSuffix = "_" + name + ".go"

var (
	typ        = flag.String("type", "", "type name; must be set")
	output     = flag.String("output", "", "output file name; default srcdir/<type>"+defaultSuffix)
	tag        = flag.String("tag", "", "tag used to constant naming")
	wrap       = flag.Bool("wrap", false, "wrap tag const by own type")
	ref        = flag.Bool("ref", false, "return field as refs in generated methods")
	export     = flag.Bool("export", false, "export generated types, constant, methods")
	exportVars = flag.Bool("exportVars", false, "export generated variables only")
)

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of "+name+":\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetPrefix(name + ": ")
	flag.Usage = Usage
	flag.Parse()

	typeName := *typ
	if len(typeName) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	args := flag.Args()
	outDirectory := outDir(args)

	pkg := extractPackage(args)
	packageName := pkg.Name
	files := pkg.Syntax
	if len(files) == 0 {
		log.Printf("no src files in package %s", packageName)
		return
	}

	typeFile := findTypeFile(files, typeName, *tag)
	if typeFile == nil {
		log.Printf("type not found, %s", typeName)
		return
	}

	g := generator.Generator{
		Name:       name,
		WrapType:   *wrap,
		ReturnRefs: *ref,
		Export:     *export,
		ExportVars: *exportVars,
	}

	g.GenerateFile(typeFile)
	src, fmtErr := g.FormatSrc()

	outputName := *output
	if outputName == "" {

		baseName := typeName + defaultSuffix
		outputName = filepath.Join(outDirectory, strings.ToLower(baseName))
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

func extractPackage(patterns []string) *packages.Package {
	packages, err := packages.Load(&packages.Config{
		Mode:  packages.NeedSyntax,
		Tests: false,
	}, patterns...)
	if err != nil {
		log.Fatal(err)
	}
	if len(packages) != 1 {
		log.Fatalf("error: %d packages found", len(packages))
	}
	return packages[0]
}

func findTypeFile(files []*ast.File, typeName string, tag string) *struc.Struct {
	for _, file := range files {
		structTags := struc.FindStructTags(file, typeName, struc.TagName(tag))
		if structTags != nil {
			return structTags
		}
	}
	return nil
}
