package generator

import (
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"
)

const DefaultMethodSuffixFromString = "FromString"

func (g *Generator) GenerateEnumFromString(typ *types.Named, constNames c.Collection[[]string],
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
		funcName    = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixFromString, name), export)
		resultVar   = "r"
		receiverVar = "s"
		body        = FuncBodyWithArgs(funcName, slice.Of(receiverVar+" string"), "("+resultVar+" "+returnType+", ok bool)", nolint, enumEvalExpr(constNames, receiverVar, resultVar))
	)
	return funcName, body, nil
}

func enumEvalExpr[C c.Collection[[]string]](consts C, receiverVar, resultVar string) string {
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
