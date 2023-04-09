package generator

import "github.com/m4gshm/fieldr/struc"

func GenerateSetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool, fieldParts []FieldInfo) string {
	buildedType := GetTypeName(model.TypeName, pkgName)
	typeName := ifElse(isReceiverReference, "*", "") + buildedType
	typeParams := TypeParamsString(model.Typ.TypeParams(), outPkgPath)
	typeParamsDecl := TypeParamsDeclarationString(model.Typ.TypeParams(), outPkgPath)

	_, conditionalPath, conditions := FiledPathAndAccessCheckCondition(receiverVar, isReceiverReference, false, fieldParts)
	varsConditionStart := ""
	varsConditionEnd := ""
	for _, c := range conditions {
		varsConditionStart += "if " + c + " {\n"
		varsConditionEnd += "}\n"
	}

	arg := LegalIdentName(IdentName(fieldName, false))
	decl := ifElse(len(pkgName) == 0,
		"func ("+receiverVar+" "+typeName+typeParams+") "+methodName+"("+arg+" "+fieldType+")",
		"func "+methodName+typeParamsDecl+"("+receiverVar+" "+typeName+typeParams+","+arg+" "+fieldType+")",
	)

	fieldMethod := decl + " {" + NoLint(nolint) + "\n" +
		varsConditionStart +
		ifElse(len(varsConditionStart) > 0, conditionalPath, receiverVar) + "." + fieldName + "=" + arg + "\n" +
		varsConditionEnd + "}\n"
	return fieldMethod
}

func GenerateGetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool, fieldParts []FieldInfo) string {
	buildedType := GetTypeName(model.TypeName, pkgName)
	typeName := ifElse(isReceiverReference, "*", "") + buildedType
	typeParams := TypeParamsString(model.Typ.TypeParams(), outPkgPath)
	typeParamsDecl := TypeParamsDeclarationString(model.Typ.TypeParams(), outPkgPath)

	emptyVar := "no"
	emptyResult := "var " + emptyVar + " " + fieldType

	_, conditionalPath, conditions := FiledPathAndAccessCheckCondition(receiverVar, isReceiverReference, false, fieldParts)

	varsConditionStart := ""
	varsConditionEnd := ""
	for _, c := range conditions {
		varsConditionStart += "if " + c + " {\n"
		varsConditionEnd += "}\n"
	}

	decl := ifElse(len(pkgName) == 0,
		"func ("+receiverVar+" "+typeName+typeParams+") "+methodName+"()",
		"func "+methodName+typeParamsDecl+"("+receiverVar+" "+typeName+typeParams+")",
	)

	fieldMethod := decl + " " + fieldType + " {" + NoLint(nolint) + "\n" +
		varsConditionStart +
		"return " + ifElse(len(varsConditionStart) > 0, conditionalPath, receiverVar) + "." + fieldName +
		varsConditionEnd + "\n" +
		ifElse(len(varsConditionStart) > 0, emptyResult+"\n"+"return "+emptyVar+"\n", "") +
		"\n}\n"

	return fieldMethod
}
