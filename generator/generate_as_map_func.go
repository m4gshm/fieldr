package generator

import (
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/replace"
	"github.com/m4gshm/gollections/op/delay/string_/wrap"
	"github.com/m4gshm/gollections/slice/split"

	"github.com/m4gshm/fieldr/model/struc"
)

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, name, keyType string,
	constants []FieldConst,
	rewriter *CodeRewriter,
	export, snake, returnRefs, noReceiver, nolint, hardcodeValues bool,
) (string, string, string, error) {

	pkgName, err := g.GetPackageNameOrAlias(model.Package().Name(), model.Package().Path())
	if err != nil {
		return "", "", "", err
	}

	typeName := model.TypeName()
	receiverVar := TypeReceiverVar(typeName)
	receiverRef := op.IfElse(returnRefs, "&"+receiverVar, receiverVar)

	mapVar := "m"
	internal := "if " + receiverVar + " == nil{\nreturn nil\n}\n" +
		mapVar + " := map[" + keyType + "]interface{}{}\n" +
		generateMapInits(g, mapVar, receiverRef, rewriter, constants) +
		"return " + mapVar

	funcName := renameFuncByConfig(IdentName("AsMap", export), name)
	typParams := model.Typ.TypeParams()
	receiverType := GetTypeName(typeName, pkgName) + TypeParamsString(typParams, g.OutPkgPath)
	returnType := "map[" + keyType + "]interface{}"
	body := MethodBody(funcName, noReceiver, receiverVar, "*"+receiverType, returnType, nolint, internal)

	return receiverType, op.IfElse(noReceiver, funcName, MethodName(typeName, funcName)), body, nil
}

func generateMapInits(g *Generator, mapVar, recVar string, rewriter *CodeRewriter, constants []FieldConst) string {
	return loop.ConvertS(constants, func(constant FieldConst) string {
		var (
			_, conditionPath, conditions         = FiledPathAndAccessCheckCondition(recVar, false, false, constant.fieldPath)
			varsConditionStart, varsConditionEnd = split.AndReduce(conditions, wrap.By("if ", " {\n"), replace.By("}\n"), op.Sum, op.Sum)
			field                                = constant.fieldPath[len(constant.fieldPath)-1]
			revr, _                              = rewriter.Transform(field.Name, field.Type, conditionPath)
		)
		return varsConditionStart + mapVar + "[" + constant.name + "]= " + revr + "\n" + varsConditionEnd
	}).Reduce(op.Sum)
}
