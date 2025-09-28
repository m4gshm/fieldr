package generator

import (
	goconstant "go/constant"
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/seq"
	"github.com/m4gshm/gollections/seq2"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/fieldr/typeparams"
)

const DefaultMethodSuffixByName = "ByName"
const DefaultMethodSuffixByValue = "ByValue"
const DefaultMethodSuffixAll = "All"

func (g *Generator) GenerateEnumFromValue(typ util.TypeNamedOrAlias, constValNamesMap c.KVRange[goconstant.Value, []string],
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
		returnType   = GetTypeName(typeName, pkgName) + typeparams.New(typParams).IdentString(g.OutPkgPath)
		funcName     = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixByValue, name), export)
		resultVar    = "e"
		receiverVar  = "value"
		receiverType = basicType.Name()
		body         = FuncBodyWithArgs(funcName, slice.Of(receiverVar+" "+receiverType), "("+resultVar+" "+returnType+", ok bool)",
			nolint, enumFromValueSwitchExpr(constValNamesMap, receiverVar, resultVar))
	)
	return funcName, body, nil
}

func enumFromValueSwitchExpr(constValNamesMap c.KVRange[goconstant.Value, []string], receiverVar, resultVar string) string {
	expr := "ok = true\n"
	expr += "switch " + receiverVar + " {\n"
	for val, names := range constValNamesMap.All {
		expr += "case " + val.ExactString()
		expr += ":\n" + "\t" + resultVar + "=" + names[0] + "\n"
	}
	expr += "default:\n\tok=false\n}"
	return expr + "\nreturn"
}

func (g *Generator) GenerateEnumFromName(typ util.TypeNamedOrAlias, constNames c.Range[[]string],
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
		returnType  = GetTypeName(typeName, pkgName) + typeparams.New(typParams).IdentString(g.OutPkgPath)
		funcName    = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixByName, name), export)
		resultVar   = "e"
		receiverVar = "name"
		body        = FuncBodyWithArgs(funcName, slice.Of(receiverVar+" string"), "("+resultVar+" "+returnType+", ok bool)", nolint, enumFromNameSwitchExpr(constNames, receiverVar, resultVar))
	)
	return funcName, body, nil
}

func enumFromNameSwitchExpr[C c.Range[[]string]](consts C, receiverVar, resultVar string) string {
	expr := "ok = true\n"
	expr += "switch " + receiverVar + " {\n"
	for names := range consts.All {
		expr += "case "
		for i, name := range names {
			expr += op.IfElse(i > 0, ",", "") + "\"" + name + "\""
		}
		expr += ":\n" + "\t" + resultVar + "=" + names[0] + "\n"
	}
	expr += "default:\n\tok=false\n}"
	return expr + "\nreturn"
}

func (g *Generator) GenerateEnumName(typ util.TypeNamedOrAlias, constValNamesMap c.KVRange[goconstant.Value, []string],
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
		returnSlice     = seq.Reduce(seq.Convert(seq2.Values(constValNamesMap.All), slice.Len), op.Max) > 1
		returnType      = op.IfElse(returnSlice, "[]string", "string")
		receiverType    = GetTypeName(typeName, pkgName) + typeparams.New(typParams).IdentString(g.OutPkgPath)
		receiverVar     = TypeReceiverVar(typeName)
		internalContent = constsSwitchExpr(constValNamesMap, receiverVar, !returnSlice)
		funcName        = IdentName(name, export)
		body            = MethodBody(funcName, false, receiverVar, receiverType, returnType, nolint, internalContent)
	)
	return MethodName(typeName, funcName), body, nil
}

func constsSwitchExpr[C c.KVRange[goconstant.Value, []string]](consts C, receiverVar string, onlyFirst bool) string {
	expr := "switch " + receiverVar + " {\n"
	for _, names := range consts.All {
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
	}
	return expr + "default:\n\treturn " + op.IfElse(onlyFirst, "\"\"", "nil") + "\n}"
}

func (g *Generator) GenerateEnumValues(typ util.TypeNamedOrAlias, constValNamesMap c.KVRange[goconstant.Value, []string],
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
		returnType = "[]" + GetTypeName(typeName, pkgName) + typeparams.New(typParams).IdentString(g.OutPkgPath)
		funcName   = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixAll, name), export)
		body       = FuncBodyNoArg(funcName, returnType, nolint, arrayExpr(constValNamesMap, returnType))
	)
	return funcName, body, nil
}

func arrayExpr[C c.KVRange[K, []string], K any](values C, returnType string) string {
	expr := returnType + "{\n"
	for _, vals := range values.All {
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
	}
	return "return " + expr + "}"
}
