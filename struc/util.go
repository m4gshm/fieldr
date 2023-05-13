package struc

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/op"

	"golang.org/x/tools/go/packages"
)

func FindTypePackageFile(typeName string, pkgs c.ForLoop[*packages.Package]) (*types.Named, *packages.Package, *ast.File, error) {
	var resultType *types.Named
	var resultPkg *packages.Package
	var resultFile *ast.File
	return resultType, resultPkg, resultFile, pkgs.For(func(pkg *packages.Package) error {
		pkgTypes := pkg.Types
		if lookup := pkgTypes.Scope().Lookup(typeName); lookup == nil {
			logger.Debugf("no type '%s' in package '%s'", typeName, pkgTypes.Name())
			return nil
		} else if structType, _, err := GetStructTypeNamed(lookup.Type()); err != nil {
			return err
		} else if structType == nil {
			return fmt.Errorf("type '%s' is not struct", typeName)
		} else {
			resultType = structType
			resultPkg = pkg
			for _, file := range pkg.Syntax {
				if lookup := file.Scope.Lookup(typeName); lookup == nil {
					logger.Debugf("no type '%s' in file '%s' package '%s'", typeName, file.Name, pkgTypes.Name())
				} else if _, ok := lookup.Decl.(*ast.TypeSpec); !ok {
					return fmt.Errorf("type '%s' is not struct in file %s", typeName, file.Name)
				} else {
					resultFile = file
					break
				}
			}
			return c.ErrBreak
		}
	})
}

func GetTypeNamed(typ types.Type) (*types.Named, int, error) {
	switch ftt := typ.(type) {
	case *types.Named:
		return ftt, 0, nil
	case *types.Pointer:
		t, p, err := GetTypeNamed(ftt.Elem())
		if err != nil {
			return nil, 0, err
		}
		return t, p + 1, nil
	default:
		return nil, 0, nil
	}
}

func GetStructTypeNamed(typ types.Type) (*types.Named, int, error) {
	if ftt, p, err := GetTypeNamed(typ); err != nil {
		return nil, 0, err
	} else if ftt != nil {
		und := ftt.Underlying()
		if _, ok := und.(*types.Struct); ok {
			return ftt, p, nil
		} else if sund, sp, err := GetStructTypeNamed(und); err != nil {
			return nil, sp + p, err
		} else if sund != nil {
			return ftt, sp + p, nil
		}
	}
	return nil, 0, nil
}

func GetStructType(t types.Type) (*types.Struct, int, error) {
	switch tt := t.(type) {
	case *types.Struct:
		return tt, 0, nil
	case *types.Pointer:
		s, pc, err := GetStructType(tt.Elem())
		if err != nil {
			return nil, 0, err
		}
		return s, pc + 1, nil
	case *types.Named:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0, nil
		}
		return GetStructType(underlying)
	case types.Type:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0, nil
		}
		return GetStructType(underlying)
	default:
		return nil, 0, nil
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
