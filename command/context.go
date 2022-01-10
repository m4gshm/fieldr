package command

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/params"
	"github.com/m4gshm/fieldr/struc"
	"github.com/m4gshm/fieldr/use"
)

type Context struct {
	Config       *params.Config
	Generator    *generator.Generator
	model        *struc.Model
	FilePackages map[*ast.File]*packages.Package
	Files        []*ast.File
	FileSet      *token.FileSet
}

func (c *Context) Model() (*struc.Model, error) {
	if m := c.model; m != nil {
		return m, nil
	}
	typ := *c.Config.Type
	if len(typ) == 0 {
		return nil, use.Err("no type arg")
	}
	model, err := struc.New(c.FilePackages, c.Files, c.FileSet, *c.Config.Type)
	if err != nil {
		return nil, err
	} else if model == nil {
		return nil, use.Err(fmt.Sprintf("type not found, %s", typ))
	}
	c.model = model
	return model, err
}
