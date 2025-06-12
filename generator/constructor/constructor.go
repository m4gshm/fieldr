package constructor

import (
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/seq"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/unique"
)

func New(name, typeName, typeParamsDecl, typeParams, receiver string, exportMethods, nolint bool, arguments, initInstace string, init func(receiver string) string) (string, string) {
	constructorName := generator.IdentName(get.If(name == generator.Autoname, sum.Of("New", typeName)).Else(name), exportMethods)

	initPart := ""
	if init != nil {
		initPart = init(receiver)
		if len(initPart) > 0 {
			initPart += "\n"
		}
	}

	body := "func " + constructorName + typeParamsDecl + "(" + arguments + ") " + "*" + typeName + typeParams + " {" + generator.NoLint(nolint) + "\n"
	createInstance := "&" + typeName + typeParams + "{ " + initInstace + " }\n"
	if len(initPart) > 0 {
		body += "r := " + createInstance
		body += initPart
		body += "return " + receiver + "\n"
	} else {
		body += "return " + createInstance
	}
	body += "}\n"
	return constructorName, body
}

func FullArgs(g *generator.Generator, model *struc.Model, constructorName string, exportMethods bool, nolint bool) (string, string, error) {
	initPart := ""
	args := ""
	for fieldName, fieldType := range model.FieldsNameAndType {
		fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
		if err != nil {
			return "", "", err
		}
		args += fieldName + " " + fullFieldType + ",\n"
		initPart += fieldName + ":" + fieldName + ",\n"
	}
	if len(initPart) > 0 {
		initPart = "\n" + initPart
	}
	if len(args) > 0 {
		args = "\n" + args
	}

	params := generator.TypeParamsSeq(model.Typ.TypeParams(), g.OutPkgPath)
	typeParamsStr := generator.TypeParamsString(params)
	typeParamsDecl := generator.TypeParamsDeclarationString(model.Typ.TypeParams(), g.OutPkgPath)
	uniqueNames := unique.NewNamesWith(unique.DistinctBySuffix("_"))
	seq.ForEach(params, uniqueNames.Add)

	name, body := New(constructorName, model.TypeName(), typeParamsDecl, typeParamsStr, uniqueNames.Get("r"), exportMethods, nolint, args, initPart, nil)
	return name, body, nil
}
