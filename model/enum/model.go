package enum

import (
	"fmt"
	"go/types"

	"github.com/m4gshm/gollections/convert"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/util"
)

type Model struct {
	typ      *types.Named
	typBasic *types.Basic
	consts   []*types.Const
}

func (m *Model) Typ() *types.Named {
	return m.typ
}

func (m *Model) Consts() []*types.Const {
	return m.consts
}

func New(outPkgPath string, typ *types.Named, scanPkg bool) (*Model, error) {
	obj := typ.Obj()
	typName := obj.Name()

	typBasic, _ := util.GetTypeBasic(typ)
	if typBasic == nil {
		return nil, fmt.Errorf("'%s' is not a basic type", typName)
	}

	rootScope := op.IfElse(scanPkg, obj.Pkg().Scope(), obj.Parent())
	extractConsts := op.IfElse(scanPkg, getConstsAll, getConstsLevel)
	consts := slice.Filter(extractConsts(rootScope), func(c *types.Const) bool { return c.Type() == typ })
	return &Model{typ: typ, typBasic: typBasic, consts: consts}, nil
}

func getConstsAll(scope *types.Scope) []*types.Const {
	var (
		consts   = getConstsLevel(scope)
		children = slice.OfIndexed(scope.NumChildren(), scope.Child)
	)
	return append(consts, slice.Flat(children, getConstsAll)...)
}

func getConstsLevel(scope *types.Scope) []*types.Const {
	objects := slice.Convert(scope.Names(), scope.Lookup)
	return slice.ConvertOK(objects, convert.ToType[*types.Const])
}
