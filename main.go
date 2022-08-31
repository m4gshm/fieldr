package main

import (
	"errors"
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
	if err := run(); err != nil {
		log.Fatal(err.Error())
	}
}

func run() error {
	log.SetPrefix(params.Name + ": ")

	config := params.NewConfig(flag.CommandLine)

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if outputDir, err := outDir(args); err != nil {
		return err
	} else if len(outputDir) > 0 {
		if err := os.Chdir(outputDir); err != nil {
			return fmt.Errorf("out chdir: %w", err)
		}
	}

	fileSet := token.NewFileSet()
	buildTags := *config.BuildTags
	pkg, err := extractPackage(fileSet, buildTags, *config.PackagePattern)
	if err != nil {
		return err
	}
	packageName := pkg.Name
	files := pkg.Syntax
	if len(files) == 0 {
		log.Printf("no src files in package %s", packageName)
		return nil
	}

	filePackages := make(map[*ast.File]*packages.Package)
	for _, file := range files {
		filePackages[file] = pkg
	}

	inputs := *config.Input

	files, err = loadSrcFiles(inputs, buildTags, fileSet, files, filePackages)
	if err != nil {
		return err
	}

	constantReplacers, err := struc.ExtractReplacers(*config.Generator.ConstReplace...)
	if err != nil {
		return err
	}
	sharedConfig, err := newFilesCommentsConfig(files, constantReplacers)
	if err != nil {
		return err
	} else if sharedConfig != nil {
		newInputs, _ := newSet(*sharedConfig.Input, inputs...)
		if len(newInputs) > 0 {
			//new inputs detected
			newFiles, err := loadSrcFiles(newInputs, buildTags, fileSet, make([]*ast.File, 0), filePackages)
			if err != nil {
				return err
			} else if additionalConfig, err := newFilesCommentsConfig(newFiles, constantReplacers); err != nil {
				return err
			} else if additionalConfig != nil {
				if sharedConfig, err = sharedConfig.MergeWith(additionalConfig, constantReplacers); err != nil {
					return err
				}
			}
			files = append(files, newFiles...)
		}
	}
	if config, err = config.MergeWith(sharedConfig, constantReplacers); err != nil {
		return err
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
	hierarchicalModel, err := struc.FindStructTags(filePackages, files, fileSet, typeName, includedTagsSet, constants, constantReplacers)
	if err != nil {
		return err
	} else if hierarchicalModel == nil || (len(hierarchicalModel.TypeName) == 0 && len(typeName) != 0) {
		log.Printf("type not found, %s", typeName)
		return nil
	} else if len(hierarchicalModel.FieldNames) == 0 {
		log.Printf("no fields in %s", typeName)
		return nil
	}

	logger.Debugw("base generating data", "model", hierarchicalModel)

	outputName := *config.Output
	if outputName == "" {
		baseName := typeName + params.DefaultFileSuffix
		outputName = strings.ToLower(baseName)
	}

	if outputName, err = filepath.Abs(outputName); err != nil {
		return err
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

	var outPkg *packages.Package
	if outFile != nil {
		outPkg = filePackages[outFile]
	} else {
		var stat os.FileInfo
		stat, err = os.Stat(outputName)
		noExists := errors.Is(err, os.ErrNotExist)
		if noExists {
			dir := filepath.Dir(outputName)
			outPkg, err = dirPackage(dir, nil)
			if err != nil {
				return err
			} else if outPkg == nil {
				return fmt.Errorf("canot detenrime output package, path '%v'", dir)
			}
		} else if err != nil {
			return err
		} else {
			if stat.IsDir() {
				return fmt.Errorf("output file is directory")
			}
			outFileSet := token.NewFileSet()
			outFile, outPkg, err = loadFile(outputName, nil, outFileSet)
			if err != nil {
				return err
			}
			if outFile != nil {
				pos := outFile.Pos()
				outFileInfo = outFileSet.File(pos)
				if outFileInfo == nil {
					return fmt.Errorf("error of reading metadata of output file %v", outputName)
				}
			}
		}
	}

	flatFields := make(map[struc.FieldName]interface{})
	flat := g.Conf.Flat
	if flat != nil {
		for _, flatField := range *flat {
			flatFields[flatField] = nil
		}
	}
	existsFlatFields := make(map[struc.FieldName]interface{})
	for _, fieldName := range hierarchicalModel.FieldNames {
		if _, nested := flatFields[fieldName]; nested {
			existsFlatFields[fieldName] = nil
		}
	}

	var model *struc.Model
	if len(existsFlatFields) > 0 {
		//make flat model
		var (
			flatFieldNames     []struc.FieldName
			flatFieldsType     = map[struc.FieldName]struc.FieldType{}
			flatFieldsTagValue = map[struc.FieldName]map[struc.TagName]struc.TagValue{}
		)
		for _, fieldName := range hierarchicalModel.FieldNames {
			if _, ok := existsFlatFields[fieldName]; ok {
				if nestedHierarchicalModel := hierarchicalModel.Nested[fieldName]; nestedHierarchicalModel != nil {
					nestedModel := nestedHierarchicalModel.Model
					for _, nestedFieldName := range nestedModel.FieldNames {
						nestedFieldRef := struc.GetFieldRef(fieldName, nestedFieldName)

						flatFieldsType[nestedFieldRef] = nestedHierarchicalModel.FieldsType[nestedFieldName]
						flatFieldsTagValue[nestedFieldRef] = nestedHierarchicalModel.FieldsTagValue[nestedFieldName]

						flatFieldNames = append(flatFieldNames, nestedFieldRef)
					}
				} else {
					flatFieldNames = append(flatFieldNames, fieldName)
				}
			} else {
				flatFieldNames = append(flatFieldNames, fieldName)
			}
			flatFieldsType[fieldName] = hierarchicalModel.FieldsType[fieldName]
			flatFieldsTagValue[fieldName] = hierarchicalModel.FieldsTagValue[fieldName]
		}

		tagsFieldValue := map[struc.TagName]map[struc.FieldName]struc.TagValue{}
		for fieldName, tagNameValues := range flatFieldsTagValue {
			for tagName, tagValue := range tagNameValues {
				fieldTagValues, ok := tagsFieldValue[tagName]
				if !ok {
					fieldTagValues = map[struc.FieldName]struc.TagValue{}
				}
				fieldTagValues[fieldName] = tagValue
				tagsFieldValue[tagName] = fieldTagValues
			}
		}

		model = &struc.Model{
			TypeName:          hierarchicalModel.TypeName,
			PackageName:       hierarchicalModel.PackageName,
			PackagePath:       hierarchicalModel.PackagePath,
			FilePath:          hierarchicalModel.FilePath,
			FieldsTagValue:    flatFieldsTagValue,
			TagsFieldValue:    tagsFieldValue,
			FieldNames:        flatFieldNames,
			FieldsType:        flatFieldsType,
			TagNames:          hierarchicalModel.TagNames,
			Constants:         hierarchicalModel.Constants,
			ConstantTemplates: hierarchicalModel.ConstantTemplates,
		}
	} else {
		model = &hierarchicalModel.Model
	}

	if err = g.GenerateFile(model, outFile, outFileInfo, outPkg); err != nil {
		return fmt.Errorf("generate file error: %s", err)
	}
	src, fmtErr := g.FormatSrc()

	const userWriteOtherRead = fs.FileMode(0644)
	if writeErr := ioutil.WriteFile(outputName, src, userWriteOtherRead); writeErr != nil {
		return fmt.Errorf("writing output: %s", writeErr)
	} else if fmtErr != nil {
		return fmt.Errorf("go src code formatting error: %s", fmtErr)
	}
	return nil
}

func newFilesCommentsConfig(files []*ast.File, constantReplacers map[string]string) (config *params.Config, err error) {
	for _, file := range files {
		if config, err = newFileCommentConfig(file, config, constantReplacers); err != nil {
			return nil, err
		}
	}
	return config, err
}

func newFileCommentConfig(file *ast.File, sharedConfig *params.Config, constantReplacers map[string]string) (*params.Config, error) {
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			commentConfig, err := newConfigComment(comment.Text)
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

func newConfigComment(text string) (*params.Config, error) {
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

func loadSrcFiles(inputs []string, buildTags []string, fileSet *token.FileSet, files []*ast.File, filePackages map[*ast.File]*packages.Package) ([]*ast.File, error) {
	for _, srcFile := range inputs {
		file, pkg, err := loadFile(srcFile, buildTags, fileSet)
		if err != nil {
			return nil, err
		}
		if _, ok := filePackages[file]; !ok {
			files = append(files, file)
			filePackages[file] = pkg
		}
	}
	return files, nil
}

func loadFile(srcFile string, buildTags []string, fileSet *token.FileSet) (*ast.File, *packages.Package, error) {
	isAbs := filepath.IsAbs(srcFile)
	if !isAbs {
		absFile, err := filepath.Abs(srcFile)
		if err != nil {
			return nil, nil, err
		}
		srcFile = absFile
	}
	file, err := parser.ParseFile(fileSet, srcFile, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	dir := filepath.Dir(srcFile)
	pkg, err := dirPackage(dir, buildTags)
	if err != nil {
		return nil, nil, err
	}
	return file, pkg, err
}

func dirPackage(dir string, buildTags []string) (*packages.Package, error) {
	pack, err := packages.Load(&packages.Config{Mode: packageMode, BuildFlags: buildTagsArg(buildTags)}, dir)
	if err != nil {
		return nil, err
	}
	for _, p := range pack {
		return p, nil
	}
	return nil, nil
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

func outDir(args []string) (string, error) {
	if len(args) > 0 {
		if dir, err := isDir(args[len(args)-1]); err != nil {
			return "", fmt.Errorf("outDir: %w", err)
		} else if dir {
			return args[len(args)-1], nil
		}
	}
	return "", nil
}

func isDir(name string) (bool, error) {
	info, err := os.Stat(name)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// const packageMode = packages.NeedSyntax | packages.NeedModule | packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedTypesInfo
const packageMode = packages.NeedSyntax | packages.NeedModule | packages.NeedName | packages.NeedTypesInfo | packages.NeedTypes

func extractPackage(fileSet *token.FileSet, buildTags []string, patterns ...string) (*packages.Package, error) {
	_packages, err := packages.Load(&packages.Config{
		Fset: fileSet, Mode: packageMode, BuildFlags: buildTagsArg(buildTags),
	}, patterns...)
	if err != nil {
		return nil, err
	}
	if len(_packages) != 1 {
		return nil, fmt.Errorf("%d packages found", len(_packages))
	}
	pack := _packages[0]
	if errs := pack.Errors; len(errs) > 0 {
		logger.Debugf("package error; %v", errs[0])
	}
	return pack, nil
}

func buildTagsArg(buildTags []string) []string {
	return []string{fmt.Sprintf("-tags=%s", strings.Join(buildTags, " "))}
}
