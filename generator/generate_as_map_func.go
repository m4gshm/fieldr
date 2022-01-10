package generator

import (
	"github.com/m4gshm/fieldr/struc"
)

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, name, keyType string,
	constants []fieldConst,
	excluded, flats map[struc.FieldName]struct{},
	rewriter *CodeRewriter,
	export, snake, returnRefs, noReceiver, nolint, hardcodeValues bool,
) (string, string, string, error) {

	pkgAlias, err := g.GetPackageAlias(model.Package.Name, model.Package.Path)
	if err != nil {
		return "", "", "", err
	}

	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, returnRefs)

	funcName := renameFuncByConfig(IdentName("AsMap", export), name)
	typeLink := GetTypeName(model.TypeName, pkgAlias)
	mapVar := "m"
	var body string
	if noReceiver {
		body = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") map[" + keyType + "]interface{}"
	} else {
		body = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() map[" + keyType + "]interface{}"
	}
	body += " {" + "\n" +
		mapVar + " := map[" + keyType + "]interface{}{}\n"

	bodyPart, err := generateMapInits(g, mapVar, receiverRef, rewriter, constants)
	if err != nil {
		return "", "", "", err
	}
	body += bodyPart
	body += "return " + mapVar + "\n" +
		"}\n"

	if !noReceiver {
		funcName = MethodName(model.TypeName, funcName)
	}
	return typeLink, funcName, body, nil
}

func generateMapInits(
	g *Generator, mapVar, receiverRef string, rewriter *CodeRewriter, constants []fieldConst,
) (string, error) {
	body := ""
	for _, constant := range constants {
		field := constant.fieldPath[len(constant.fieldPath)-1]
		condition := ""
		fieldPath := ""
		for _, p := range constant.fieldPath {
			if len(fieldPath) > 0 {
				fieldPath += "."
			}
			fieldPath += p.name
			if p.typ.Ref {
				if len(condition) > 0 {
					condition += " && "
				}
				condition += receiverRef + "." + fieldPath + " != nil "
			}
		}
		revr, _ := rewriter.Transform(field.name, field.typ, receiverRef+"."+fieldPath)
		if len(condition) > 0 {
			body += "if " + condition + " {\n"
		}
		body += mapVar + "[" + constant.name + "]= " + revr + "\n"
		if len(condition) > 0 {
			body += "}\n"
		}

	}
	return body, nil
}
