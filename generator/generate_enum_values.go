package generator

import (
	"go/types"

	"github.com/m4gshm/gollections/c"
	oset "github.com/m4gshm/gollections/collection/immutable/ordered/set"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"

	"github.com/m4gshm/fieldr/model/enum"
)

func (g *Generator) GenerateEnumValues(model *enum.Model, name string, export bool, nolint bool) (string, string, error) {
	typ := model.Typ()
	obj := typ.Obj()
	pkg := obj.Pkg()

	typeName := obj.Name()
	typParams := typ.TypeParams()

	pkgName, err := g.GetPackageNameOrAlias(pkg.Name(), pkg.Path())
	if err != nil {
		return "", "", err
	}
	returnType := "[]" + GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)

	var (
		consts   = oset.From(loop.Convert(loop.Of(model.Consts()...), (*types.Const).Name))
		funcName = IdentName(op.IfElse(name == Autoname, typeName+"Values", name), export)
		body     = FuncBodyNoArg(funcName, returnType, nolint, ArrayExpr(returnType, consts))
	)
	return funcName, body, nil
}

func ArrayExpr[C c.ForEach[string]](returnType string, values C) string {
	expr := returnType + "{\n"
	values.ForEach(func(val string) { expr += val + ",\n" })
	return "return " + expr + "}"
}
