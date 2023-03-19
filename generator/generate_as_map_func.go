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

func TypeArgsString(targs *types.TypeList, basePkgPath string) string {
	return loopTypes(targs, basePkgPath, (*types.TypeList).Len, (*types.TypeList).At)
}

func TypeParamsString(tparams *types.TypeParamList, basePkgPath string) string {
	return loopTypes(tparams, basePkgPath, (*types.TypeParamList).Len, (*types.TypeParamList).At)
}

func loopTypes[L any, T types.Type](list L, basePkgPath string, len func(L) int, at func(L, int) T) string {
	l := len(list)
	if l == 0 {
		return ""
	}
	s := "["
	for i := 0; i < l; i++ {
		elem := at(list, i)
		if i > 0 {
			s += ", "
		}
		var n types.Type = elem
		if n == nil {
			s += "/*error: nil type parameter*/"
			continue
		}
		s += struc.TypeString(elem, basePkgPath)
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
		fullFieldPath, condition := FiledPathAndAccessCheckCondition(recVar, false, constant.fieldPath)
		revr, _ := rewriter.Transform(field.Name, field.Type, fullFieldPath)
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
