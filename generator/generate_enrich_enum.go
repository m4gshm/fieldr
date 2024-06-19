package generator

import (
	goconstant "go/constant"
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/collection/immutable/ordered"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
)

const DefaultMethodSuffixByName = "ByName"
const DefaultMethodSuffixByValue = "ByValue"
const DefaultMethodSuffixAll = "All"

func (g *Generator) GenerateEnumFromValue(typ *types.Named, constValNamesMap ordered.Map[goconstant.Value, []string],
	name string, export bool, nolint bool) (string, string, error) {

	obj := typ.Obj()
	pkg := obj.Pkg()

	pkgName, err := g.GetPackageNameOrAlias(pkg.Name(), pkg.Path())
	if err != nil {
		return "", "", err
	}

	typeName := obj.Name()
	typParams := typ.TypeParams()

	var baseType types.Type = typ
	for {
		next := baseType.Underlying()
		if next != nil && next != baseType {
			baseType = next
		} else {
			break
		}
	}
	basicType, _ := baseType.(*types.Basic)

	var (
		returnType   = GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
		funcName     = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixByValue, name), export)
		resultVar    = "e"
		receiverVar  = "value"
		receiverType = basicType.Name()
		body         = FuncBodyWithArgs(funcName, slice.Of(receiverVar+" "+receiverType), "("+resultVar+" "+returnType+", ok bool)",
			nolint, enumFromValueSwitchExpr(constValNamesMap, receiverVar, resultVar))
	)
	return funcName, body, nil
}

func enumFromValueSwitchExpr(constValNamesMap ordered.Map[goconstant.Value, []string], receiverVar, resultVar string) string {
	expr := "ok = true\n"
	expr += "switch " + receiverVar + " {\n"
	constValNamesMap.TrackEach(func(val goconstant.Value, names []string) {
		expr += "case " + val.ExactString()
		expr += ":\n" + "\t" + resultVar + "=" + names[0] + "\n"
	})
	expr += "default:\n\tok=false\n}"
	return expr + "\nreturn"
}

func (g *Generator) GenerateEnumFromName(typ *types.Named, constNames c.Collection[[]string],
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
		returnType  = GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
		funcName    = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixByName, name), export)
		resultVar   = "e"
		receiverVar = "name"
		body        = FuncBodyWithArgs(funcName, slice.Of(receiverVar+" string"), "("+resultVar+" "+returnType+", ok bool)", nolint, enumFromNameSwitchExpr(constNames, receiverVar, resultVar))
	)
	return funcName, body, nil
}

func enumFromNameSwitchExpr[C c.Collection[[]string]](consts C, receiverVar, resultVar string) string {
	expr := "ok = true\n"
	expr += "switch " + receiverVar + " {\n"
	consts.ForEach(func(names []string) {
		expr += "case "
		for i, name := range names {
			expr += op.IfElse(i > 0, ",", "") + "\"" + name + "\""
		}
		expr += ":\n" + "\t" + resultVar + "=" + names[0] + "\n"
	})
	expr += "default:\n\tok=false\n}"
	return expr + "\nreturn"
}

func (g *Generator) GenerateEnumName(typ *types.Named, constValNamesMap ordered.Map[goconstant.Value, []string],
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

func (g *Generator) GenerateEnumValues(typ *types.Named, constValNamesMap collection.Map[goconstant.Value, []string],
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
		returnType = "[]" + GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
		funcName   = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixAll, name), export)
		body       = FuncBodyNoArg(funcName, returnType, nolint, arrayExpr(constValNamesMap, returnType))
	)
	return funcName, body, nil
}

func arrayExpr[C c.TrackEach[K, []string], K any](values C, returnType string) string {
	expr := returnType + "{\n"
	values.TrackEach(func(_ K, vals []string) {
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
