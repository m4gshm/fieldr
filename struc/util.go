package struc

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/map_"
	"github.com/m4gshm/gollections/op"
	"golang.org/x/tools/go/packages"

	"github.com/m4gshm/fieldr/logger"
)

func FindTypePackageFile(typeName string, fileSet *token.FileSet, pkgs c.For[*packages.Package]) (*types.Named, *packages.Package, *ast.File, error) {
	var resultType *types.Named
	var resultPkg *packages.Package
	var resultFile *ast.File
	return resultType, resultPkg, resultFile, pkgs.For(func(pkg *packages.Package) error {
		pkgTypes := pkg.Types
		if lookup := pkgTypes.Scope().Lookup(typeName); lookup == nil {
			logger.Debugf("no type '%s' in package '%s'", typeName, pkgTypes.Name())
			return nil
		} else if structType, _ := GetStructTypeNamed(lookup.Type()); structType == nil {
			return fmt.Errorf("type '%s' is not struct", typeName)
		} else {
			resultType = structType
			resultPkg = pkg
			logger.Debugf("look package '%s', syntax file count %d", pkg.Name, len(pkg.Syntax))
			for _, file := range pkg.Syntax {
				if tokenFile := fileSet.File(file.Pos()); tokenFile != nil {
					fileName := tokenFile.Name()
					logger.Debugf("file by position '%d', name %s", file.Pos(), fileName)
					if lookup := file.Scope.Lookup(typeName); lookup == nil {
						types := map_.Keys(file.Scope.Objects)
						logger.Debugf("no type '%s' in file '%s', package '%s', types %#v", typeName, fileName, pkgTypes.Name(), types)
					} else if _, ok := lookup.Decl.(*ast.TypeSpec); !ok {
						return fmt.Errorf("type '%s' is not struct in file '%s'", typeName, fileName)
					} else {
						resultFile = file
						logger.Debugf("found type file '%s'", fileName)
						break
					}
				}
			}
			return c.Break
		}
	})
}

func GetTypeNamed(typ types.Type) (*types.Named, int) {
	switch ftt := typ.(type) {
	case *types.Named:
		return ftt, 0
	case *types.Pointer:
		t, p := GetTypeNamed(ftt.Elem())
		return t, p + 1
	default:
		return nil, 0
	}
}

func GetStructTypeNamed(typ types.Type) (*types.Named, int) {
	if ftt, p := GetTypeNamed(typ); ftt != nil {
		und := ftt.Underlying()
		if _, ok := und.(*types.Struct); ok {
			return ftt, p
		} else if sund, sp := GetStructTypeNamed(und); sund != nil {
			return ftt, sp + p
		}
	}
	return nil, 0
}

func GetStructType(t types.Type) (*types.Struct, int) {
	switch tt := t.(type) {
	case *types.Struct:
		return tt, 0
	case *types.Pointer:
		s, pc := GetStructType(tt.Elem())
		return s, pc + 1
	case *types.Named:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0
		}
		return GetStructType(underlying)
	case types.Type:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0
		}
		return GetStructType(underlying)
	default:
		return nil, 0
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
