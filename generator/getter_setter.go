package generator

import (
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/replace"
	"github.com/m4gshm/gollections/op/delay/string_"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/slice/split"

	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"
)

func GenerateSetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool, fieldParts []FieldInfo) string {

	uniqueNames := unique.NewNamesWith(unique.PreInit(receiverVar), unique.DistinctBySuffix("_"))
	params := typeparams.New(model.Typ.TypeParams())
	params.Names(outPkgPath).ForEach(uniqueNames.Add)

	buildedType := GetTypeName(model.TypeName(), pkgName)
	typeName := op.IfElse(isReceiverReference, "*", "") + buildedType
	typeParams, typeParamsDecl := params.IdentDeclStrings(outPkgPath)
	_, conditionalPath, conditions := FiledPathAndAccessCheckCondition(receiverVar, isReceiverReference, false, fieldParts, uniqueNames)
	varsConditionStart, varsConditionEnd := split.AndReduce(conditions, string_.Wrap("if ", " {\n"), replace.By("}\n"), op.Sum, op.Sum)

	arg := uniqueNames.Get(LegalIdentName(ArgName(fieldName)))
	return get.If(len(pkgName) == 0,
		sum.Of("func (", receiverVar, " ", typeName, typeParams, ") ", methodName, "(", arg, " ", fieldType, ")")).ElseGet(
		sum.Of("func ", methodName, typeParamsDecl, "(", receiverVar, " ", typeName, typeParams, ",", arg, " ", fieldType, ")"),
	) + " {" + NoLint(nolint) + "\n" + varsConditionStart +
		op.IfElse(len(varsConditionStart) > 0, conditionalPath, receiverVar) + "." + fieldName + "=" + arg + "\n" +
		varsConditionEnd + "}\n"
}

func GenerateGetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool, fieldParts []FieldInfo) string {
	uniqueNames := unique.NewNamesWith(unique.PreInit(receiverVar), unique.DistinctBySuffix("_"))
	params := typeparams.New(model.Typ.TypeParams())
	params.Names(outPkgPath).ForEach(uniqueNames.Add)

	buildedType := GetTypeName(model.TypeName(), pkgName)
	typeName := op.IfElse(isReceiverReference, "*", "") + buildedType
	typeParams, typeParamsDecl := params.IdentDeclStrings(outPkgPath)
	_, conditionalPath, conditions := FiledPathAndAccessCheckCondition(receiverVar, isReceiverReference, false, fieldParts, uniqueNames)
	varsConditionStart, varsConditionEnd := split.AndReduce(conditions, string_.Wrap("if ", " {\n"), replace.By("}\n"), op.Sum, op.Sum)

	emptyVar := uniqueNames.Get("no")
	emptyResult := "var " + emptyVar + " " + fieldType

	return get.If(len(pkgName) == 0,
		sum.Of("func (", receiverVar, " ", typeName, typeParams, ") ", methodName, "()")).ElseGet(
		sum.Of("func ", methodName, typeParamsDecl, "(", receiverVar, " ", typeName, typeParams, ")"),
	) + " " + fieldType + " {" + NoLint(nolint) + "\n" + varsConditionStart + "return " +
		op.IfElse(len(varsConditionStart) > 0, conditionalPath, receiverVar) + "." + fieldName +
		varsConditionEnd + "\n" + get.If(len(varsConditionStart) > 0, sum.Of(emptyResult, "\n", "return ", emptyVar, "\n")).Else("") + "}\n"
}
