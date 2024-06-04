package generator

import (
	goconstant "go/constant"
	"go/types"
	"strings"

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

	internalContent := ConstsSwitchExpr(conctValNamesMap, !returnSlice)
	typParams := model.Typ().TypeParams()
	receiverType := GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
	returnType := op.IfElse(returnSlice, "[]string", "string")
	body := FuncBody(funcName, false, receiverVar, receiverType, returnType, nolint, internalContent)
	return MethodName(typeName, funcName), body, nil
}

func ConstsSwitchExpr[C collection.Map[goconstant.Value, []string]](consts C, onlyFirst bool) string {
	varName := "e"

	switchCaseBody := strings.Builder{}
	switchCaseBody.WriteString("switch ")
	switchCaseBody.WriteString(varName)
	switchCaseBody.WriteString(" {\n")

	for val, names := range consts.All {
		vs := val.ExactString()
		switchCaseBody.WriteString("case ")
		switchCaseBody.WriteString(vs)
		switchCaseBody.WriteString(":\n")
		switchCaseBody.WriteString("\treturn")
		if onlyFirst {
			switchCaseBody.WriteString("\"")
			switchCaseBody.WriteString(names[0])
			switchCaseBody.WriteString("\"")
		} else {
			switchCaseBody.WriteString("[]string{")
			for i, name := range names {
				if i > 0 {
					switchCaseBody.WriteString(",")
				}
				switchCaseBody.WriteString("\"")
				switchCaseBody.WriteString(name)
				switchCaseBody.WriteString("\"")
			}
			switchCaseBody.WriteString("}")
		}
		switchCaseBody.WriteString("\n")
	}

	switchCaseBody.WriteString("default:\n\treturn ")
	if onlyFirst {
		switchCaseBody.WriteString("\"\"")
	} else {
		switchCaseBody.WriteString("nil")
	}
	switchCaseBody.WriteString("\n}")

	s := switchCaseBody.String()
	return s
}
