package generator

import (
	"go/types"

	"github.com/m4gshm/gollections/c"
	ordermap "github.com/m4gshm/gollections/collection/immutable/ordered/map_"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice/group"

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
		constValNamesMap = ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
		funcName         = IdentName(op.IfElse(name == Autoname, typeName+"Values", name), export)
		body             = FuncBodyNoArg(funcName, returnType, nolint, arrayExpr(returnType, constValNamesMap.Values()))
	)
	return funcName, body, nil
}

func arrayExpr[C c.ForEach[[]string]](returnType string, values C) string {
	expr := returnType + "{\n"
	values.ForEach(func(vals []string) {
		for i, val := range vals {
			if i == 0 {
				expr += val + ","
			} else if i == 1 {
				expr += " //" + val
			} else {
				expr += ", " + val
			}
		}
		expr += "\n"
	})

	return "return " + expr + "}"
}
