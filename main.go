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
	"path"
	"path/filepath"
	"strings"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
	"golang.org/x/tools/go/packages"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of "+params.Name+":\n")
	fmt.Fprintf(os.Stderr, "\t"+params.Name+" [flags] -type T [directory]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetPrefix(params.Name + ": ")

	config := params.NewConfig(flag.CommandLine)

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	outputDir := outDir(args)
	if len(outputDir) > 0 {
		if err := os.Chdir(outputDir); err != nil {
			log.Fatalf("out dir error: %v", err)
		}
	}

	fileSet := token.NewFileSet()
	pkg := extractPackage(fileSet, *config.BuildTags, *config.PackagePattern)
	packageName := pkg.Name
	files := pkg.Syntax
	if len(files) == 0 {
		log.Printf("no src files in package %s", packageName)
		return
	}

	inputs := *config.Input
	var err error
	files, err = loadSrcFiles(inputs, fileSet, files)
	if err != nil {
		log.Fatal(err)
	}
	constantReplacers, err := struc.ExtractReplacers(*config.Generator.ConstReplace...)
	if err != nil {
		log.Fatal(err)
	}
	sharedConfig, err := NewFilesCommentsConfig(files, constantReplacers)
	if err != nil {
		log.Fatal(err)
	}
	if sharedConfig != nil {
		newInputs, _ := newSet(*sharedConfig.Input, inputs...)
		if len(newInputs) > 0 {
			//new inputs detected
			newFiles, err := loadSrcFiles(newInputs, fileSet, make([]*ast.File, 0))
			if err != nil {
				log.Fatal(err)
			} else if additionalConfig, err := NewFilesCommentsConfig(newFiles, constantReplacers); err != nil {
				log.Fatal(err)
			} else if additionalConfig != nil {
				if sharedConfig, err = sharedConfig.MergeWith(additionalConfig, constantReplacers); err != nil {
					log.Fatal(err)
				}
			}
			files = append(files, newFiles...)
		}
	}
	if config, err = config.MergeWith(sharedConfig, constantReplacers); err != nil {
		log.Fatal(err)
	}

	logger.Debugw("using", "config", config)

	typeName := *config.Type
	if len(typeName) == 0 {
		log.Print("no type arg")
		flag.Usage()
		os.Exit(2)
	}

	var (
		includedTagArg  = *config.Generator.IncludeFieldTags
		includedTagsSet = make(map[struc.TagName]interface{})
		includedTags    = make([]struc.TagName, 0)
	)
	if len(includedTagArg) > 0 {
		includedTagNames := strings.Split(includedTagArg, ",")
		for _, includedTag := range includedTagNames {
			name := struc.TagName(includedTag)
			includedTagsSet[name] = nil
			includedTags = append(includedTags, name)
		}
	}
	constants := *config.Content.Constants
	structModel, err := struc.FindStructTags(files, typeName, includedTagsSet, constants, constantReplacers)
	if err != nil {
		log.Fatal(err)
	} else if structModel == nil || (len(structModel.TypeName) == 0 && len(typeName) != 0) {
		log.Printf("type not found, %s", typeName)
		return
	} else if len(structModel.FieldNames) == 0 {
		log.Printf("no fields in %s", typeName)
		return
	}

	logger.Debugw("base generating data", "model", structModel)

	outputName := *config.Output
	if outputName == "" {
		baseName := typeName + params.DefaultFileSuffix
		outputName = strings.ToLower(baseName)
	}

	if outputName, err = filepath.Abs(outputName); err != nil {
		log.Fatal(err)
	}

	var outFile *ast.File
	var outFileInfo *token.File
	for _, file := range files {
		if info := fileSet.File(file.Pos()); info.Name() == outputName {
			outFileInfo = info
			outFile = file
			break
		}
	}

	g := &generator.Generator{
		IncludedTags: includedTags,
		Name:         params.Name,
		Conf:         config.Generator,
		Content:      config.Content,
	}

	if err = g.GenerateFile(structModel, outFile, outFileInfo); err != nil {
		log.Fatalf("generate file error: %s", err)
	}
	src, fmtErr := g.FormatSrc()

	const userWriteOtherRead = fs.FileMode(0644)
	if writeErr := ioutil.WriteFile(outputName, src, userWriteOtherRead); writeErr != nil {
		log.Fatalf("writing output: %s", writeErr)
	} else if fmtErr != nil {
		log.Fatalf("go src code formatting error: %s", fmtErr)
	}
}

func NewFilesCommentsConfig(files []*ast.File, constantReplacers map[string]string) (config *params.Config, err error) {
	for _, file := range files {
		if config, err = NewFileCommentConfig(file, config, constantReplacers); err != nil {
			return nil, err
		}
	}
	return config, err
}

func NewFileCommentConfig(file *ast.File, sharedConfig *params.Config, constantReplacers map[string]string) (*params.Config, error) {
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			commentConfig, err := NewConfigComment(comment.Text)
			if err != nil {
				return nil, err
			} else if sharedConfig == nil {
				sharedConfig = commentConfig
				continue
			} else if sharedConfig, err = sharedConfig.MergeWith(commentConfig, constantReplacers); err != nil {
				return nil, err
			}
		}
	}
	return sharedConfig, nil
}

func NewConfigComment(text string) (*params.Config, error) {
	prefix := "//" + params.CommentConfigPrefix
	if len(text) > 0 && strings.HasPrefix(text, prefix) {
		configComment := text[len(prefix)+1:]
		if len(configComment) > 0 {
			flagSet := flag.NewFlagSet(params.CommentConfigPrefix, flag.ExitOnError)
			commentConfig := params.NewConfig(flagSet)
			var err error
			if err = flagSet.Parse(strings.Split(configComment, " ")); err != nil {
				return nil, fmt.Errorf("parsing cofig comment %v; %w", text, err)
			}

			return commentConfig, nil
		}
	}
	return nil, nil
}

func loadSrcFiles(inputs []string, fileSet *token.FileSet, files []*ast.File) ([]*ast.File, error) {
	for _, srcFile := range inputs {
		file, err := loadFile(srcFile, fileSet)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func loadFile(srcFile string, fileSet *token.FileSet) (*ast.File, error) {
	isAbs := path.IsAbs(srcFile)
	if !isAbs {
		absFile, err := filepath.Abs(srcFile)
		if err != nil {
			return nil, err
		}
		srcFile = absFile
	}
	return parser.ParseFile(fileSet, srcFile, nil, parser.ParseComments)
}

var emptySet = map[string]int{}
var emptySlice []string

func newSet(values []string, excludes ...string) ([]string, map[string]int) {
	if len(values) == 0 {
		return emptySlice, emptySet
	}
	uniques := make([]string, 0)
	_, exclSet := newSet(excludes)
	set := make(map[string]int)
	for i, value := range values {
		if _, ok := exclSet[value]; !ok {
			if _, ok = set[value]; !ok {
				set[value] = i
				uniques = append(uniques, value)
			}
		}
	}
	return uniques, set
}

func outDir(args []string) string {
	if len(args) > 0 && isDir(args[len(args)-1]) {
		return args[len(args)-1]
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

func extractPackage(fileSet *token.FileSet, buildTags []string, patterns ...string) *packages.Package {
	packages, err := packages.Load(&packages.Config{
		Fset:       fileSet,
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
