package generator

import (
	"strconv"

	"github.com/m4gshm/gollections/op"

	"github.com/m4gshm/fieldr/model/struc"
)

func GenerateOptionFieldFunc(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool,
	isReceiverReference bool, fieldParts []FieldInfo) string {
	buildedType := GetTypeName(model.TypeName(), pkgName)
	typeName := op.IfElse(isReceiverReference, "*", "") + buildedType
	typeParams := TypeParamsString(model.Typ.TypeParams(), outPkgPath)
	typeParamsDecl := TypeParamsDeclarationString(model.Typ.TypeParams(), outPkgPath)
	accessInfo := GetFieldConditionalPartsAccessInfo(receiverVar, fieldParts)
	variableName := accessInfo.ShortVar

	funcBody := ""
	for _, fc := range accessInfo.AccessPathParts {
		shortVar := fc.ShortVar
		newExpr := generateNewObjectExpr(shortVar, fc.Type.RefDeep, fc.Type.FullName(outPkgPath))
		newIfNilExpr := shortVar + " := " + fc.FieldPath + "\nif " + shortVar + " == nil " + "{\n" + newExpr + "\n" + fc.FieldPath + " = " + shortVar + "}\n"
		funcBody += newIfNilExpr
	}

	arg := LegalIdentName(IdentName(fieldName, false))
	optType := "func (" + receiverVar + " " + typeName + typeParams + ")"

	result := "func " + methodName + typeParamsDecl + "(" + arg + " " + fieldType + ") " + optType +
		" {" + NoLint(nolint) + "\n" + "return " + optType + " {" + funcBody +
		variableName + "." + fieldName + "=" + arg + "\n" + "}\n" + "}\n"
	return result
}

func generateNewObjectExpr(receiverVariable string, refDeep int, valTypeSign string) string {
	if refDeep > 0 {
		valTypeSign = valTypeSign[refDeep:]
	}
	var newExpr string
	deepRefCount := refDeep - 1
	if deepRefCount <= 0 {
		newExpr = receiverVariable + " = " + "new(" + valTypeSign + ")"
	} else {
		newExpr = receiverVariable + strconv.Itoa(deepRefCount) + " := " + "new(" + valTypeSign + ")"
		for r := deepRefCount - 1; r >= 0; r-- {
			newExpr += "\n" + receiverVariable + strconv.Itoa(r) + " := &" + receiverVariable + strconv.Itoa(r+1)
		}
		newExpr += "\n" + receiverVariable + " = &" + receiverVariable + strconv.Itoa(0)
	}
	return newExpr
}
