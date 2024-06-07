package generator

import (
	goconstant "go/constant"
	"go/types"

	"github.com/m4gshm/gollections/collection"
	ordermap "github.com/m4gshm/gollections/collection/immutable/ordered/map_"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
	"github.com/m4gshm/gollections/slice/group"

	"github.com/m4gshm/fieldr/model/enum"
)

func (g *Generator) GenerateEnumStringify(model *enum.Model, name string, export bool, nolint bool) (string, string, error) {
	typ := model.Typ()
	obj := typ.Obj()
	pkg := obj.Pkg()

	pkgName, err := g.GetPackageNameOrAlias(pkg.Name(), pkg.Path())
	if err != nil {
		return "", "", err
	}
	typeName := obj.Name()
	receiverVar := TypeReceiverVar(typeName)
	funcName := IdentName(name, export)

	conctValNamesMap := ordermap.New(group.Order(model.Consts(), (*types.Const).Val, (*types.Const).Name))
	maxConstsPerVal := loop.Reduce(collection.Convert(conctValNamesMap.Values(), slice.Len), op.Max)
	returnSlice := maxConstsPerVal > 1

	internalContent := ConstsSwitchExpr(conctValNamesMap, receiverVar, !returnSlice)
	typParams := model.Typ().TypeParams()
	receiverType := GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
	returnType := op.IfElse(returnSlice, "[]string", "string")
	body := FuncBody(funcName, false, receiverVar, receiverType, returnType, nolint, internalContent)
	return MethodName(typeName, funcName), body, nil
}

func ConstsSwitchExpr[C collection.Map[goconstant.Value, []string]](consts C, receiverVar string, onlyFirst bool) string {
	expr := "switch " + receiverVar + " {\n"
	consts.TrackEach(func(val goconstant.Value, names []string) {
		expr += "case " + val.ExactString() + ":\n" + "\treturn"
		if onlyFirst {
			expr += "\"" + names[0] + "\""
		} else {
			expr += "[]string{"
			for i, name := range names {
				expr += op.IfElse(i > 0, ",", "") + "\"" + name + "\""
			}
			expr += "}"
		}
		expr += "\n"
	})
	return expr + "default:\n\treturn " + op.IfElse(onlyFirst, "\"\"", "nil") + "\n}"
}
