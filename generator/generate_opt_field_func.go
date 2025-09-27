package generator

import (
	"go/types"
	"strconv"

	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"
	"github.com/m4gshm/gollections/seq"
)

func GenerateOptionFieldFunc(model *struc.Model, pkgName, receiverVar, methodName, fieldName, fieldType, outPkgPath string, nolint bool, fieldParts []FieldInfo) string {
	typeName := "*" + GetTypeName(model.TypeName(), pkgName)
	params := typeparams.New(model.Typ.TypeParams())
	typeParams := params.IdentString(outPkgPath)
	typeParamsDecl := params.DeclarationString(outPkgPath)
	uniqueNames := unique.NewNamesWith(unique.PreInit(receiverVar))
	seq.ForEach(params.Names(outPkgPath), uniqueNames.Add)

	accessInfo := GetFieldConditionalPartsAccessInfo(receiverVar, fieldParts, uniqueNames)
	variableName := accessInfo.ShortVar

	funcBody := ""
	for _, accessPathPart := range accessInfo.AccessPathParts {
		shortVar := accessPathPart.ShortVar
		typ := accessPathPart.Type.Type
		newExpr := generateNewObjectExpr(typ, outPkgPath, shortVar)
		newIfNilExpr := shortVar + " := " + accessPathPart.FieldPath + "\nif " + shortVar + " == nil " + "{\n" + newExpr + "\n" + accessPathPart.FieldPath + " = " + shortVar + "}\n"
		funcBody += newIfNilExpr
	}

	arg := uniqueNames.Get(LegalIdentName(ArgName(fieldName)))
	optType := "func (" + receiverVar + " " + typeName + typeParams + ")"

	result := "func " + methodName + typeParamsDecl + "(" + arg + " " + fieldType + ") " + optType +
		" {" + NoLint(nolint) + "\n" + "return " + optType + " {" + funcBody +
		variableName + "." + fieldName + "=" + arg + "\n" + "}\n" + "}\n"
	return result
}

func generateNewObjectExpr(typ types.Type, outPkgPath string, receiverVariable string) string {
	valType, refDeep := util.GetTypeUnderPointer(typ)
	valTypeName := util.TypeString(valType, outPkgPath)
	deepRefCount := refDeep - 1
	if deepRefCount <= 0 {
		return receiverVariable + " = " + "new(" + valTypeName + ")"
	}
	newExpr := receiverVariable + strconv.Itoa(deepRefCount) + " := " + "new(" + valTypeName + ")"
	for r := deepRefCount - 1; r >= 0; r-- {
		newExpr += "\n" + receiverVariable + strconv.Itoa(r) + " := &" + receiverVariable + strconv.Itoa(r+1)
	}
	newExpr += "\n" + receiverVariable + " = &" + receiverVariable + strconv.Itoa(0)
	return newExpr
}
