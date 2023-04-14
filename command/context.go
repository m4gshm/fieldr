package command

import (
	"go/types"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/struc"
	"github.com/m4gshm/fieldr/use"
)

type Context struct {
	Generator *generator.Generator
	model     *struc.Model
	Typ       *types.Named
	Pkg       struc.Package
}

func (c *Context) Model() (*struc.Model, error) {
	if m := c.model; m != nil {
		return m, nil
	}
	if c.Typ == nil {
		logger.Debugf("error config without type")
		return nil, use.Err("no type arg")
	}

	model, err := struc.New(c.Generator.OutPkgPath, c.Typ, c.Pkg)
	c.model = model
	return model, err
}
