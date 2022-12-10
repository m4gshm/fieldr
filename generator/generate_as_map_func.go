package generator

import (
	"go/types"

	"github.com/m4gshm/fieldr/struc"
	"github.com/m4gshm/gollections/c"
)

func (g *Generator) GenerateAsMapFunc(
	model *struc.Model, name, keyType string,
	constants []fieldConst,
	/*excluded map[struc.FieldName]struct{}, */ flats c.Set[string],
	rewriter *CodeRewriter,
	export, snake, returnRefs, noReceiver, nolint, hardcodeValues bool,
) (string, string, string, error) {

	pkgName, err := g.GetPackageName(model.Package.Name, model.Package.Path)
	if err != nil {
		return "", "", "", err
	}

	receiverVar := "v"
	receiverRef := AsRefIfNeed(receiverVar, returnRefs)

	funcName := renameFuncByConfig(IdentName("AsMap", export), name)

	typeLink := GetTypeName(model.TypeName, pkgName) + TypeParamsString(model.Typ.TypeParams(), g.OutPkg.PkgPath)
	mapVar := "m"
	var body string
	if noReceiver {
		body = "func " + funcName + "(" + receiverVar + " *" + typeLink + ") map[" + keyType + "]interface{}"
	} else {
		body = "func (" + receiverVar + " *" + typeLink + ") " + funcName + "() map[" + keyType + "]interface{}"
	}
	body += " {" + NoLint(nolint)
	body += "\n"

	body += "if " + receiverVar + " == nil{\nreturn nil\n}\n"

	body += mapVar + " := map[" + keyType + "]interface{}{}\n"

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

func TypeParamsString(tparams *types.TypeParamList, basePkgPath string) string {
	l := tparams.Len()
	if l == 0 {
		return ""
	}
	s := "["
	for i := 0; i < l; i++ {
		tpar := tparams.At(i)
		if i > 0 {
			s += ", "
		}
		if tpar == nil {
			s += "/*error: nil type parameter*/"
			continue
		}
		s += struc.TypeString(tpar, basePkgPath)
	}
	s += "]"
	return s
}

func generateMapInits(
	g *Generator, mapVar, recVar string, rewriter *CodeRewriter, constants []fieldConst,
) (string, error) {
	body := ""
	for _, constant := range constants {
		field := constant.fieldPath[len(constant.fieldPath)-1]
		fullFieldPath, condition := FiledPathAndAcceddCheckCondition(recVar, constant.fieldPath)
		revr, _ := rewriter.Transform(field.name, field.typ, fullFieldPath)
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
