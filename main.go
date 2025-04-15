package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io/fs"
	"io/ioutil"
	"iter"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	breakloop "github.com/m4gshm/gollections/break/loop"
	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/mutable/ordered"
	"github.com/m4gshm/gollections/collection/mutable/ordered/map_"
	"github.com/m4gshm/gollections/collection/mutable/ordered/set"
	"github.com/m4gshm/gollections/error_"
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
	"golang.org/x/tools/go/packages"

	"github.com/m4gshm/fieldr/command"
	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/fieldr/params"
	fuse "github.com/m4gshm/fieldr/use"
)

func usage(commandLine *flag.FlagSet) func() {
	return func() {
		out := commandLine.Output()
		_, _ = fmt.Fprintf(out, params.Name+" is a tool for generating constants, variables, functions and methods"+
			" based on a type properties like name, structure fields, tags or base type nature.\n")
		_, _ = fmt.Fprintf(out, "Usage of "+params.Name+":\n")
		_, _ = fmt.Fprintf(out, "\t"+params.Name+" [flags] command1 [command-flags] command2 [command-flags]... command [command-flags]\n")
		_, _ = fmt.Fprintf(out, "Use \"command --help\" to get help of this one\n")
		_, _ = fmt.Fprintf(out, "Flags:\n")
		commandLine.PrintDefaults()
		_, _ = fmt.Fprintf(out, " --help\n")
		_, _ = fmt.Fprintf(out, "\tshow this message\n")
		command.PrintUsage()
	}
}

func main() {
	if err := run(); err != nil {
		var uErr *fuse.Error
		if errors.As(err, &uErr) {
			fmt.Fprintf(os.Stderr, "err: "+uErr.Error()+"\n")
			flag.CommandLine.Usage()
		} else {
			log.Fatal(err.Error())
		}
	}
}

func run() error {
	appFile, appArgs := os.Args[0], os.Args[1:]

	configParser := newConfigFlagSet(appFile)
	flag.CommandLine = configParser

	debugFlag := configParser.Bool("debug", false, "enable debug logging")
	buildTags := params.MultiVal(configParser, "buildTag", []string{"fieldr"}, "include build tag")
	inputs := params.InFlag(configParser)
	packageSearchPath := configParser.String("path", "", "search packages path")

	commonTypeConfig := params.NewTypeConfig(configParser)
	if err := configParser.Parse(appArgs); err != nil {
		return fmt.Errorf("parse args: %v: %w", appArgs, err)
	}

	logger.Init(*debugFlag)
	logger.Debugf("common type config: type '%v', output '%v'", commonTypeConfig.Type, commonTypeConfig.Output)

	configParserArgs := configParser.Args()
	commands, args, err := parseCommands(configParserArgs)
	if err != nil {
		return fmt.Errorf("parse commands: %v: %w", configParserArgs, err)
	}
	if len(commands) == 0 {
		logger.Debugf("no command line generator commands")
	}
	if len(args) > 0 {
		logger.Debugf("unspent command line args %v\n", args)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workdir: %w", err)
	}

	fileSet := token.NewFileSet()

	pkgs, err := extractPackages(fileSet, *buildTags, workDir)
	if err != nil {
		return fmt.Errorf("extract packages, workDir %s, build tags %v: %w", workDir, *buildTags, err)
	}

	typeConfig := *commonTypeConfig
	notCmdLineType := len(typeConfig.Type) == 0

	typeConfigs := map_.Empty[params.TypeConfig, []*command.Command]()

	for file, err := range getFilesCommentArgs(fileSet, getAstFiles(pkgs)) {
		if err != nil {
			return err
		}
		for _, commentCmd := range file.CommentArgs() {
			configParser := newConfigFlagSet(strings.Join(commentCmd.args, " "))
			commentConfig := params.NewTypeConfig(configParser)
			if err = configParser.Parse(commentCmd.args); err != nil {
				return err
			}
			if notCmdLineType {
				if len(commentConfig.Type) != 0 {
					typeConfig.Type = commentConfig.Type
					if len(typeConfig.Output) == 0 {
						typeConfig.Output = commentConfig.Output
					}
					if len(typeConfig.OutPackage) == 0 {
						typeConfig.OutPackage = commentConfig.OutPackage
					}
					if len(typeConfig.OutBuildTags) == 0 {
						typeConfig.OutBuildTags = commentConfig.OutBuildTags
					}
					logger.Debugf("init first type %+v by comment type %+v", typeConfig, *commentConfig)
				}
				notCmdLineType = false
			}

			if commentConfig.Type == typeConfig.Type && commentConfig.Output == typeConfig.Output {
				logger.Debugf("skip comment config because its type and out are equal to prev: comment config %+v, prev %+v", commentConfig, typeConfig)
				//skip
			} else if len(commentConfig.Type) == 0 && commentConfig.Output == typeConfig.Output {
				//skip
				logger.Debugf("skip comment config because its out is equal to prev: comment config %+v, prev %+v", commentConfig, typeConfig)
			} else if len(commentConfig.Type) != 0 || len(commentConfig.Output) != 0 {
				if len(commentConfig.Type) == 0 {
					(*commentConfig).Type = typeConfig.Type
				}

				logger.Debugf("detect another type %+v\n", *commentConfig)

				if len(commands) == 0 {
					logger.Debugf("no commands for type %v", typeConfig)
					typeConfig = *commentConfig
				} else {
					typeConfigs.Set(typeConfig, commands)
					logger.Debugf("set type %+v, commands %d\n", typeConfig, len(commands))
					typeConfig = *commentConfig
					commands = []*command.Command{}
				}
			}

			cmtCommands, cmtArgs, err := parseCommands(configParser.Args())
			if uErr, ok := error_.As[*fuse.Error](err); ok {
				return fuse.FileCommentErr(uErr.Error(), file.astFile, file.tokenFile, commentCmd.comment)
			} else if err != nil {
				return err
			} else if len(cmtCommands) == 0 {
				// logger.Debugf("no comment generator commands: file %s, line: %d args %v\n", f.file.Name, cmt.comment.Pos(), cmtArgs)
			} else if len(cmtArgs) > 0 {
				logger.Debugf("unspent comment line args: %v\n", cmtArgs)
			}
			commands = append(commands, cmtCommands...)
		}
	}

	typeConfigs.Set(typeConfig, commands)

	if pkgPtrn := *packageSearchPath; len(pkgPtrn) > 0 {
		if patternPkgs, err := extractPackages(fileSet, *buildTags, pkgPtrn); err != nil {
			return err
		} else {
			pkgs.AddAllNew(patternPkgs)
		}
	}

	logger.Debugf("set type last %+v, commands: %s\n", typeConfig, strings.Join(slice.Convert(commands, (*command.Command).Name), ", "))

	if inputPkgs, err := loadFilesPackages(fileSet, *inputs, *buildTags); err != nil {
		return err
	} else {
		pkgs.AddAllNew(inputPkgs)
	}

	if logger.IsDebug() {
		pkgsFiles := getAstFiles(pkgs)
		logger.Debugf("source files amount %d", pkgsFiles.Len())

		for file := range pkgsFiles.All {
			if info := fileSet.File(file.Pos()); info != nil {
				logger.Debugf("found source file %s", info.Name())
			}
		}
	}

	for typeConfig, commands := range typeConfigs.All {
		logger.Debugf("using type config %+v\n", typeConfig)

		typeName := typeConfig.Type

		if len(typeName) == 0 {
			logger.Debugf("error config without type %+v", typeConfig)
			return fuse.Err("no type arg")
		}

		typ, typPkg, typFile, err := util.FindTypePackageFile(typeName, fileSet, pkgs)
		if err != nil {
			return fmt.Errorf("find type %s: %w", typeName, err)
		} else if typ == nil {
			return fmt.Errorf("type not found: %s", typeName)
		}

		outputName, err := get.If(typeConfig.Output == generator.Autoname, func() string {
			outFileInfo := fileSet.File(typFile.Pos())
			autoselected := outFileInfo.Name()
			logger.Debugf("autoselected out file '%s'", autoselected)
			return autoselected
		}).ElseGetErr(func() (string, error) {
			out := typeConfig.Output
			return abs(op.IfElse(len(out) > 0, out, strings.ToLower(typeName+params.DefaultFileSuffix)))
		})
		if err != nil {
			return err
		}

		logger.Debugf("output file %s", outputName)
		if typPkg == nil {
			return fmt.Errorf("type package not found: type %s", typeName)
		}
		typModule := typPkg.Module
		moduleDir := typModule.Dir

		outPkg, outFile, outFileInfo, err := findPkgFile(fileSet, pkgs, outputName, moduleDir)
		if err != nil {
			return err
		}

		if outFile == nil {
			logger.Debugf("out file not found, trying to fix")
			buildTag := typeConfig.OutBuildTags

			dir, err := getDir(outputName)
			if err != nil {
				return err
			} else if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
				logger.Debugf("create new package dir %s", dir)
				if err := os.Mkdir(dir, os.ModePerm); err != nil {
					return err
				}
			}

			if outPkgs, err := loadFilePackage(dir, fileSet, buildTag); err != nil {
				return err
			} else if outPkg, outFile, outFileInfo, err = findPkgFile(fileSet, outPkgs, outputName, moduleDir); err != nil {
				return fmt.Errorf("findPkgFile: out file %s :%w", outputName, err)
			} else if outPkg == nil {
				logger.Debugf("cannot determine output package, create new: output file '%s', moduleDir '%s', dir '%s'", outputName, moduleDir, dir)

				pkgPath, err := filepath.Rel(moduleDir, dir)
				if err != nil {
					return err
				}
				pkgPath = op.IfElse(pkgPath == ".", "", pkgPath)
				pkgName := op.IfElse(pkgPath != "", filepath.Base(pkgPath), "")
				typs := types.NewPackage(pkgPath, pkgName)
				outPkg = &packages.Package{PkgPath: pkgPath, ID: pkgPath, Name: pkgName, Types: typs, Module: typModule}
				logger.Debugf("create package type %#v", outPkg)
			}
		}

		if outPkg == nil {
			return fmt.Errorf("out package is undefined")
		}

		var pkgTypes *types.Package
		var pkgPath string
		if outPkg != nil {
			pkgTypes = outPkg.Types
			pkgPath = outPkg.PkgPath
		}
		g := generator.New(params.Name, typeConfig.OutBuildTags, outFile, outFileInfo, pkgPath, pkgTypes)
		o := typ.Obj()
		pp := o.Pkg()
		_ = pp
		ctx := &command.Context{Generator: g, Typ: typ}
		for _, c := range commands {
			logger.Debugf("run command %s", c.Name())
			if err := c.Run(ctx); err != nil {
				return fmt.Errorf("run: %w", err)
			}
		}

		outPackageName := generator.OutPackageName(typeConfig.OutPackage, outPkg)
		if err := g.WriteBody(outPackageName); err != nil {
			return fmt.Errorf("write body, outPackageName %s: %w", outPackageName, err)
		}

		src, fmtErr := g.FormatSrc()

		const userWriteOtherRead = fs.FileMode(0644)
		if writeErr := ioutil.WriteFile(outputName, src, userWriteOtherRead); writeErr != nil {
			return fmt.Errorf("writing output: %s", writeErr)
		} else if fmtErr != nil {
			return fmt.Errorf("go src code formatting error: %s", fmtErr)
		}
	}
	return nil
}

func getPkgFiles(p *packages.Package) []*ast.File { return p.Syntax }

func getAstFiles(pkgs *ordered.Set[*packages.Package]) *ordered.Set[*ast.File] {
	return set.From(collection.Flat(pkgs, getPkgFiles))
}

func findPkgFile(fileSet *token.FileSet, pkgs *ordered.Set[*packages.Package], outputName, moduleDir string) (*packages.Package, *ast.File, *token.File, error) {
	logger.Debugf("findPkgFile: outputName %s", outputName)

	for pkg, file := range loop.ExtraVals(pkgs.Loop(), getPkgFiles).All {
		if info := fileSet.File(file.Pos()); info != nil {
			srcFileName := info.Name()
			if srcFileName == outputName {
				logger.Debugf("finPkgFile: file found %s", outputName)
				return pkg, file, info, nil
			}
			logger.Debugf("findPkgFile: looked file %s", srcFileName)
		}
	}

	logger.Debugf("findPkgFile: output file not found: %s", outputName)

	dir, err := getDir(outputName)
	if err != nil {
		return nil, nil, nil, err
	}
	logger.Debugf("findPkgFile: find package by exist src files")

	for pkg, file := range loop.ExtraVals(pkgs.Loop(), getPkgFiles).All {
		if info := fileSet.File(file.Pos()); info != nil {
			if fileDir, err := getDir(info.Name()); err != nil {
				return nil, nil, nil, err
			} else if fileDir == dir {
				logger.Debugf("findPkgFile: found package '%s' by file '%s'", pkg.Name, info.Name())
				return pkg, nil, nil, nil
			}
		}
	}

	logger.Debugf("findPkgFile: cannot determine package  exist src files")

	pkgName := filepath.Base(dir)
	logger.Debugf("findPkgFile: select package by name: %s, path %s", pkgName, dir)
	firstPkg, ok := collection.First(pkgs, func(p *packages.Package) bool {
		if fullPath := filepath.Join(p.Module.Dir, path.Base(p.PkgPath)); fullPath == dir {
			logger.Debugf("findPkgFile: found package by name %s, %v", pkgName, fullPath)
			return true
		} else {
			logger.Debugf("findPkgFile: not match package by name: %s, path %s", p.PkgPath, fullPath)
			return false
		}
	})

	if !ok {
		logger.Debugf("findPkgFile: package not found by name %s", pkgName)
	}
	return firstPkg, nil, nil, nil

}

func newConfigFlagSet(name string) *flag.FlagSet {
	configParser := flag.NewFlagSet(name, flag.ExitOnError)
	configParser.Usage = usage(configParser)
	return configParser
}

func parseCommands(args []string) ([]*command.Command, []string, error) {
	commands := []*command.Command{}
	for len(args) > 0 {
		cmd, cmdArgs := args[0], args[1:]
		if c := command.Get(cmd); c == nil {
			return nil, args, fuse.Err("unknowd command '" + cmd + "'")
		} else if unusedArgs, err := c.Parse(cmdArgs); err != nil {
			return nil, nil, err
		} else {
			args = unusedArgs
			commands = append(commands, c)
		}
	}
	return commands, args, nil
}

type fileCommentArgs struct {
	astFile     *ast.File
	tokenFile   *token.File
	commentArgs []commentArgs
}

func (f fileCommentArgs) CommentArgs() []commentArgs { return f.commentArgs }

func getFilesCommentArgs(fileSet *token.FileSet, files collection.Iterable[*ast.File]) iter.Seq2[fileCommentArgs, error] {
	return breakloop.NoNilPtrVal(collection.Conv(files, func(file *ast.File) (*fileCommentArgs, error) {
		ft := fileSet.File(file.Pos())
		if args, err := getCommentArgs(file, ft); err != nil {
			return nil, err
		} else if len(args) > 0 {
			return &fileCommentArgs{astFile: file, tokenFile: ft, commentArgs: args}, nil
		}
		return nil, nil
	})).All
}

type commentArgs struct {
	comment *ast.Comment
	args    []string
}

func getCommentArgs(file *ast.File, fInfo *token.File) ([]commentArgs, error) {
	return loop.Conv(loop.FlatS(file.Comments, func(cg *ast.CommentGroup) []*ast.Comment { return cg.List }),
		func(comment *ast.Comment) (a commentArgs, err error) {
			args, err := getCommentCmdArgs(comment.Text)
			if err == nil && len(args) > 0 {
				logger.Debugf("extracted comment args: file %s, line %d, args %v", fInfo.Name(), fInfo.Line(comment.Pos()), args)
			}
			return commentArgs{comment: comment, args: args}, err
		},
	).Slice()
}

var commentCmdPrefix = "//" + params.CommentConfigPrefix

func getCommentCmdArgs(text string) ([]string, error) {
	if len(text) > 0 && strings.HasPrefix(text, commentCmdPrefix) {
		if configComment := text[len(commentCmdPrefix)+1:]; len(configComment) > 0 {
			logger.Debugf("split comment args '%s'", configComment)
			if args, err := splitArgs(configComment); err != nil {
				return nil, fmt.Errorf("split cofig comment %v; %w", text, err)
			} else {
				logger.Debugf("comment args count %d, '%s'", len(args), strings.Join(args, ","))
				return args, nil
			}
		}
	}
	return nil, nil
}

func splitArgs(rawArgs string) ([]string, error) {
	var args []string
	for {
		rawArgs = strings.TrimLeft(rawArgs, " ")
		if len(rawArgs) == 0 {
			break
		}
		symbols := []rune(rawArgs)
		if symbols[0] == '"' {
			finished := false
			//start parsing quoted string
		quoted:
			for i := 1; i < len(symbols); i++ {
				c := symbols[i]
				switch c {
				case '\\':
					if i+1 == len(symbols) {
						return nil, errors.New("unexpected backslash at the end")
					}
					i++
				case '"':
					part := rawArgs[0 : i+1]
					arg, err := strconv.Unquote(part)
					if err != nil {
						return nil, fmt.Errorf("unquote string: %s: %w", part, err)
					}
					args = append(args, arg)
					rawArgs = string(symbols[i+1:])
					//finish parsing quoted string
					finished = true
					break quoted
				}
			}
			if !finished {
				return nil, errors.New("unclosed quoted string")
			}
		} else {
			i := strings.Index(rawArgs, " ")
			if i < 0 {
				i = len(rawArgs)
			}
			args = append(args, rawArgs[0:i])
			rawArgs = rawArgs[i:]
		}
	}
	return args, nil
}

func loadFilesPackages(fileSet *token.FileSet, inputs []string, buildTags []string) (*ordered.Set[*packages.Package], error) {
	return loop.ConvS(inputs, func(srcFile string) (*ordered.Set[*packages.Package], error) {
		return loadFilePackage(srcFile, fileSet, buildTags...)
	}).Reducee(func(l, r *ordered.Set[*packages.Package]) (*ordered.Set[*packages.Package], error) {
		_ = l.AddAllNew(r)
		return l, nil
	})
}

func abs(srcFile string) (string, error) {
	if !filepath.IsAbs(srcFile) {
		a, err := filepath.Abs(srcFile)
		if err != nil {
			return a, fmt.Errorf("absolue file: %s: %w", srcFile, err)
		}
		return a, nil
	}
	return srcFile, nil
}

func loadFilePackage(srcFile string, fileSet *token.FileSet, buildTags ...string) (*ordered.Set[*packages.Package], error) {
	absSrcFile, err := abs(srcFile)
	if err != nil {
		return nil, err
	}
	return extractPackages(fileSet, buildTags, absSrcFile)
}

const packageMode = packages.NeedSyntax | packages.NeedName | packages.NeedTypesInfo | packages.NeedTypes | packages.NeedModule

func extractPackages(fileSet *token.FileSet, buildTags []string, fileName string) (*ordered.Set[*packages.Package], error) {
	if dir, err := getDir(fileName); err != nil {
		return nil, err
	} else if pkgs, err := packages.Load(&packages.Config{
		Dir:        dir,
		Fset:       fileSet,
		Mode:       packageMode,
		BuildFlags: buildTagsArg(buildTags),
		Tests:      true,
		Logf:       func(format string, args ...interface{}) { logger.Debugf("packagesLoad: "+format, args...) },
	}, "."); err != nil {
		return nil, err
	} else {
		return set.Of(pkgs...), nil
	}
}
func getDir(fileName string) (string, error) {
	fileStat, err := os.Stat(fileName)
	isNoExists := errors.Is(err, os.ErrNotExist)
	if !isNoExists && err != nil {
		return "", err
	}
	return use.If(!isNoExists && fileStat.IsDir(), fileName).ElseGet(func() string { return filepath.Dir(fileName) }), nil
}

func buildTagsArg(buildTags []string) []string {
	return []string{fmt.Sprintf("-tags=%s", strings.Join(buildTags, " "))}
}
