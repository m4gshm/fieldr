package generator

import "github.com/m4gshm/fieldr/struc"

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, pkg, name string,
	rewriter *CodeRewriter,
	export, snake, wrapType, returnRefs, noReceiver, allFields, nolint, hardcodeValues bool,
) (string, string, string, error) {

	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, returnRefs)

	keyType := BaseConstType
	if wrapType {
		g.used.fieldType = true
		keyType = getUsedFieldType(model.TypeName, export, snake)
	}

	funcName := renameFuncByConfig(goName("AsMap", export), name)
	typeLink := getTypeName(model.TypeName, pkg)
	var funcBody string
	if noReceiver {
		funcBody = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") map[" + keyType + "]interface{}"
	} else {
		funcBody = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() map[" + keyType + "]interface{}"
	}
	funcBody += " {" + g.noLint(nolint) + "\n" +
		"	return map[" + keyType + "]interface{}{\n"

	for _, fieldName := range model.FieldNames {
		if g.isFieldExcluded(fieldName, allFields) {
			continue
		}
		funcBody += g.getUsedFieldConstName(model.TypeName, fieldName, hardcodeValues, export, snake) + ": " +
			rewriter.Transform(fieldName, model.FieldsType[fieldName], struc.GetFieldRef(receiverRef, fieldName)) + ",\n"
	}
	funcBody += "" +
		"	}\n" +
		"}\n"
	return typeLink, funcName, funcBody, nil
}
