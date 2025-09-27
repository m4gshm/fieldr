package command

import (
	"go/ast"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/enum"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/fieldr/use"
)

type Context struct {
	Generator   *generator.Generator
	structModel *struc.Model
	enumModel   *enum.Model
	Typ         util.TypeNamedOrAlias
	TypFile     *ast.File
}

func (c *Context) StructModel() (*struc.Model, error) {
	if m := c.structModel; m != nil {
		return m, nil
	}
	if c.Typ == nil {
		logger.Debugf("error config without type")
		return nil, use.Err("no type in context")
	}

	model, err := struc.New(c.Generator.OutPkgPath, c.Typ, c.TypFile)
	c.structModel = model
	return model, err
}

func (c *Context) EnumModel() (*enum.Model, error) {
	if m := c.enumModel; m != nil {
		return m, nil
	}
	if c.Typ == nil {
		logger.Debugf("error config without type")
		return nil, use.Err("no type in context")
	}

	model, err := enum.New(c.Generator.OutPkgPath, c.Typ, false)
	c.enumModel = model
	return model, err
}
