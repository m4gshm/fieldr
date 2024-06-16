package generator

import (
	goconstant "go/constant"
	"go/types"

	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/immutable/ordered"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
)

func (g *Generator) GenerateEnumStringify(typ *types.Named, constValNamesMap ordered.Map[goconstant.Value, []string],
	name string, export bool, nolint bool) (string, string, error) {

	obj := typ.Obj()
	pkg := obj.Pkg()

	pkgName, err := g.GetPackageNameOrAlias(pkg.Name(), pkg.Path())
	if err != nil {
		return "", "", err
	}

	typeName := obj.Name()
	typParams := typ.TypeParams()

	var (
		returnSlice     = loop.Reduce(collection.Convert(constValNamesMap.Values(), slice.Len), op.Max) > 1
		returnType      = op.IfElse(returnSlice, "[]string", "string")
		receiverType    = GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
		receiverVar     = TypeReceiverVar(typeName)
		internalContent = constsSwitchExpr(constValNamesMap, receiverVar, !returnSlice)
		funcName        = IdentName(name, export)
		body            = MethodBody(funcName, false, receiverVar, receiverType, returnType, nolint, internalContent)
	)
	return MethodName(typeName, funcName), body, nil
}

func constsSwitchExpr[C collection.Map[goconstant.Value, []string]](consts C, receiverVar string, onlyFirst bool) string {
	expr := "switch " + receiverVar + " {\n"
	consts.TrackEach(func(val goconstant.Value, names []string) {
		expr += "case " + names[0] + ":\n" + "\treturn"
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
