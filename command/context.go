package command

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
	"github.com/m4gshm/fieldr/use"
	"github.com/m4gshm/gollections/immutable/ordered"
)

type Context struct {
	TypeConfig   params.TypeConfig
	Generator    *generator.Generator
	model        *struc.Model
	FilePackages ordered.Map[*ast.File, *packages.Package]
	FileSet      *token.FileSet
}

func (c *Context) Model() (*struc.Model, error) {
	if m := c.model; m != nil {
		return m, nil
	}
	typ := c.TypeConfig.Type
	if len(typ) == 0 {
		logger.Debugf("error config without type %v", c.TypeConfig)
		return nil, use.Err("no type arg")
	}
	model, err := struc.New(c.Generator.OutPkg.PkgPath, c.FilePackages, c.FileSet, typ)
	if err != nil {
		return nil, err
	} else if model == nil {
		return nil, use.Err(fmt.Sprintf("type not found: %s", typ))
	}
	c.model = model
	return model, err
}
