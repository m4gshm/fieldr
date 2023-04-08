package generator

import "github.com/m4gshm/fieldr/struc"

func GenerateSetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool) string {
	buildedType := GetTypeName(model.TypeName, pkgName)
	typeName := ifElse(isReceiverReference, "*", "") + buildedType
	typeParams := TypeParamsString(model.Typ.TypeParams(), outPkgPath)
	typeParamsDecl := TypeParamsDeclarationString(model.Typ.TypeParams(), outPkgPath)

	arg := LegalIdentName(IdentName(fieldName, false))
	decl := ifElse(len(pkgName) == 0,
		"func ("+receiverVar+" "+typeName+typeParams+") "+methodName+"("+arg+" "+fieldType+")",
		"func "+methodName+typeParamsDecl+"("+receiverVar+" "+typeName+typeParams+","+arg+" "+fieldType+")",
	)

	fieldMethod := decl + " {" + NoLint(nolint) + "\n" +
		ifElse(isReceiverReference, "if "+receiverVar+" != nil {\n", "") +
		receiverVar + "." + fieldName + "=" + arg + "\n" +
		ifElse(isReceiverReference, "}\n", "") + "}\n"
	return fieldMethod
}

func GenerateGetter(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, isReceiverReference bool) string {
	buildedType := GetTypeName(model.TypeName, pkgName)
	typeName := ifElse(isReceiverReference, "*", "") + buildedType
	typeParams := TypeParamsString(model.Typ.TypeParams(), outPkgPath)
	typeParamsDecl := TypeParamsDeclarationString(model.Typ.TypeParams(), outPkgPath)

	emptyVar := "no"
	emptyResult := "var " + emptyVar + " " + fieldType

	decl := ifElse(len(pkgName) == 0,
		"func ("+receiverVar+" "+typeName+typeParams+") "+methodName+"()",
		"func "+methodName+typeParamsDecl+"("+receiverVar+" "+typeName+typeParams+")",
	)
	fieldMethod := decl + " " + fieldType + " {" + NoLint(nolint) + "\n" +
		ifElse(isReceiverReference, "if "+receiverVar+" != nil {\n", "") +
		emptyResult + "\n" + "return " + emptyVar + "\n" +
		ifElse(isReceiverReference, "}\n", "") +
		"return " + receiverVar + "." + fieldName + "\n}\n"
	return fieldMethod
}
