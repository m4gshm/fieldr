package generator

import (
	"go/types"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/loop/convert"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/replace"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/delay/string_/wrap"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/slice/split"

	"github.com/m4gshm/fieldr/struc"
)

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, name, keyType string,
	constants []FieldConst,
	rewriter *CodeRewriter,
	export, snake, returnRefs, noReceiver, nolint, hardcodeValues bool,
) (string, string, string, error) {

	pkgName, err := g.GetPackageName(model.Package.Name, model.Package.Path)
	if err != nil {
		return "", "", "", err
	}

	receiverVar := TypeReceiverVar(model.TypeName)
	receiverRef := op.IfElse(returnRefs, "&"+receiverVar, receiverVar)

	funcName := renameFuncByConfig(IdentName("AsMap", export), name)

	typeLink := GetTypeName(model.TypeName, pkgName) + TypeParamsString(model.Typ.TypeParams(), g.OutPkgPath)
	mapVar := "m"

	body := "func " + get.If(noReceiver,
		sum.Of(funcName, "(", receiverVar, " *", typeLink, ")"),
	).ElseGet(
		sum.Of("(", receiverVar, " *", typeLink, ") ", funcName, "()"),
	) + " map[" + keyType + "]interface{}" + " {" + NoLint(nolint) + "\n" +
		"if " + receiverVar + " == nil{\nreturn nil\n}\n" +
		mapVar + " := map[" + keyType + "]interface{}{}\n" +
		generateMapInits(g, mapVar, receiverRef, rewriter, constants) +
		"return " + mapVar + "\n}\n"

	return typeLink, op.IfElse(noReceiver, funcName, MethodName(model.TypeName, funcName)), body, nil
}

func TypeParamsString(tparams *types.TypeParamList, basePkgPath string) string {
	return string_.WrapNonEmpty("[", loop.Reduce(convert.FromIndexed(tparams.Len(), tparams.At, func(elem *types.TypeParam) string {
		return use.If(elem == nil, "/*error: nil type parameter*/").ElseGet(
			func() string { return struc.TypeString(elem, basePkgPath) })
	}), join.NonEmpty(", ")), "]")
}

func TypeParamsDeclarationString(list *types.TypeParamList, basePkgPath string) string {
	var (
		prevElem types.Type
		noFirst  = false
	)
	return string_.WrapNonEmpty("[", loop.Reduce(convert.FromIndexed(list.Len(), list.At, func(elem *types.TypeParam) string {
		s := use.If(elem == nil, "/*error: nil type parameter*/").ElseGet(func() string {
			constraint := elem.Constraint()
			s := use.If(!noFirst, "").IfGet(constraint != prevElem, sum.Of(" ", struc.TypeString(prevElem, basePkgPath), ",")).Else(",")
			prevElem = constraint
			return s + struc.TypeString(elem, basePkgPath)
		})
		noFirst = true
		return s
	}), op.Sum[string])+get.If(prevElem != nil, func() string { return " " + struc.TypeString(prevElem, basePkgPath) }).Else(""), "]")
}

func generateMapInits(g *Generator, mapVar, recVar string, rewriter *CodeRewriter, constants []FieldConst) string {
	return loop.Convert(loop.Of(constants...), func(constant FieldConst) string {
		var (
			_, conditionPath, conditions         = FiledPathAndAccessCheckCondition(recVar, false, false, constant.fieldPath)
			varsConditionStart, varsConditionEnd = split.AndReduce(conditions, wrap.By("if ", " {\n"), replace.By("}\n"), op.Sum[string], op.Sum[string])
			field                                = constant.fieldPath[len(constant.fieldPath)-1]
			revr, _                              = rewriter.Transform(field.Name, field.Type, conditionPath)
		)
		return varsConditionStart + mapVar + "[" + constant.name + "]= " + revr + "\n" + varsConditionEnd
	}).Reduce(op.Sum)
}
