package generator

import (
	goconstant "go/constant"
	"go/types"

	"github.com/m4gshm/gollections/c"
	"github.com/m4gshm/gollections/collection"
	"github.com/m4gshm/gollections/op"
)

const DefaultMethodSuffixValues = "Values"

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
		funcName   = IdentName(op.IfElse(name == Autoname, typeName+DefaultMethodSuffixValues, name), export)
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
