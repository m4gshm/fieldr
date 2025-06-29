package util

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/collection/mutable/ordered"
	"github.com/m4gshm/gollections/collection/mutable/ordered/set"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/map_"
	"github.com/m4gshm/gollections/op"
	"golang.org/x/tools/go/packages"

	"github.com/m4gshm/fieldr/logger"
)

const packageMode = packages.NeedSyntax | packages.NeedName | packages.NeedTypesInfo | packages.NeedTypes | packages.NeedModule | packages.NeedForTest

func ExtractPackages(fileSet *token.FileSet, buildTags []string, fileName string) (*ordered.Set[*packages.Package], error) {
	if dir, err := GetDir(fileName); err != nil {
		return nil, err
	} else if pkgs, err := packages.Load(&packages.Config{
		Dir:        dir,
		Fset:       fileSet,
		Mode:       packageMode,
		BuildFlags: buildTagsArg(buildTags),
		Tests:      true,
		Logf:       func(format string, args ...any) { logger.Debugf("packagesLoad: "+format, args...) },
	}, "."); err != nil {
		return nil, err
	} else {
		return set.Of(pkgs...), nil
	}
}

func buildTagsArg(buildTags []string) []string {
	return []string{fmt.Sprintf("-tags=%s", strings.Join(buildTags, " "))}
}

func GetDir(fileName string) (string, error) {
	fileStat, err := os.Stat(fileName)
	isNoExists := errors.Is(err, os.ErrNotExist)
	if !isNoExists && err != nil {
		return "", err
	}
	return use.If(!isNoExists && fileStat.IsDir(), fileName).ElseGet(func() string { return filepath.Dir(fileName) }), nil
}

func FindTypePackageFile(typeName string, fileSet *token.FileSet, pkgs c.Range[*packages.Package]) (TypeNamedOrAlias, *packages.Package, *ast.File, error) {
	for pkg := range pkgs.All {
		pkgTypes := pkg.Types
		if lookup := pkgTypes.Scope().Lookup(typeName); lookup == nil {
			logger.Debugf("no type '%s' in package '%s'", typeName, pkgTypes.Name())
			continue
		} else if typeNamed, _ := GetTypeNamed(lookup.Type()); typeNamed == nil {
			return nil, nil, nil, fmt.Errorf("cannot detect type '%s'", typeName)
		} else {
			var resultFile *ast.File
			logger.Debugf("look package '%s', syntax file count %d", pkg.Name, len(pkg.Syntax))
			for _, file := range pkg.Syntax {
				if tokenFile := fileSet.File(file.Pos()); tokenFile != nil {
					fileName := tokenFile.Name()
					logger.Debugf("file by position '%d', name %s", file.Pos(), fileName)
					if lookup := file.Scope.Lookup(typeName); lookup == nil {
						types := map_.Keys(file.Scope.Objects)
						logger.Debugf("no type '%s' in file '%s', package '%s', types %#v", typeName, fileName, pkgTypes.Name(), types)
					} else if _, ok := lookup.Decl.(*ast.TypeSpec); !ok {
						return nil, nil, nil, fmt.Errorf("type '%s' is not struct in file '%s'", typeName, fileName)
					} else {
						resultFile = file
						logger.Debugf("found type file '%s'", fileName)
						break
					}
				}
			}
			return typeNamed, pkg, resultFile, nil
		}
	}
	return nil, nil, nil, nil
}

func GetTypeUnderPointer(typ types.Type) (types.Type, int) {
	switch ftt := typ.(type) {
	case *types.Pointer:
		t, p := GetTypeUnderPointer(ftt.Elem())
		return t, p + 1
	case *types.Named, *types.Alias:
		t, p := GetTypeUnderPointer(ftt.Underlying())
		if p == 0 {
			return ftt, 0
		} 
		return t, p
	default:
		return typ, 0
	}
}

type TypeNamedOrAlias interface {
	types.Type
	Underlying() types.Type
	Obj() *types.TypeName
	TypeParams() *types.TypeParamList
}

var _ TypeNamedOrAlias = (*types.Named)(nil)
var _ TypeNamedOrAlias = (*types.Alias)(nil)
var _ types.Type = (TypeNamedOrAlias)(nil)

func GetTypeNamed(typ types.Type) (TypeNamedOrAlias, int) {
	switch ftt := typ.(type) {
	case *types.Named:
		return ftt, 0
	case *types.Alias:
		return ftt, 0
	case *types.Pointer:
		t, p := GetTypeNamed(ftt.Elem())
		return t, p + 1
	default:
		return nil, 0
	}
}

func GetStructTypeNamed(typ types.Type) (TypeNamedOrAlias, int) {
	if ftt, deep := GetTypeNamed(typ); ftt != nil {
		und := ftt.Underlying()
		if _, ok := und.(*types.Struct); ok {
			return ftt, deep
		} else if sund, sp := GetStructTypeNamed(und); sund != nil {
			return ftt, sp + deep
		}
	}
	return nil, 0
}

func GetTypeStruct(t types.Type) (*types.Struct, int) {
	return getType[*types.Struct](t, 1000)
}

func GetTypeBasic(t types.Type) (*types.Basic, int) {
	return getType[*types.Basic](t, 1000)
}

func getType[T types.Type](t types.Type, depth int) (T, int) {
	if depth < 0 {
		panic(fmt.Sprintf("getType overflow %v", t))
	}
	var zero T
	switch tt := t.(type) {
	case T:
		return tt, 0
	case *types.Pointer:
		s, pc := getType[T](tt.Elem(), depth-1)
		return s, pc + 1
	case types.Type:
		underlying := tt.Underlying()
		return getType[T](underlying, depth-1)
	default:
		return zero, 0
	}
}

func TypeString(typ types.Type, outPkgPath string) string {
	return types.TypeString(typ, basePackQ(outPkgPath))
}

func ObjectString(obj types.Object, outPkgPath string) string {
	return types.ObjectString(obj, basePackQ(outPkgPath))
}

func basePackQ(outPkgPath string) func(p *types.Package) string {
	return func(p *types.Package) string {
		return op.IfElse(p.Path() == outPkgPath, "", p.Name())
	}
}
