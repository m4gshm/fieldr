package constructor

import (
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/seq"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"
)

func New(
	name, typeName, typeParamsDecl, typeParams, receiver string,
	returnVal, exportMethods, nolint bool,
	arguments, initInstace string, init func(receiver string) string,
) (string, string) {
	constructorName := generator.IdentName(get.If(name == generator.Autoname, sum.Of("New", typeName)).Else(name), exportMethods)

	initPart := ""
	if init != nil {
		initPart = init(receiver)
		if len(initPart) > 0 {
			initPart += "\n"
		}
	}

	body := "func " + constructorName + typeParamsDecl + "(" + arguments + ") " + op.IfElse(returnVal, "", "*") +
		typeName + typeParams + " {" + generator.NoLint(nolint) + "\n"
	createInstance := op.IfElse(returnVal, "", "&") + typeName + typeParams + "{ " + initInstace + " }\n"
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

func FullArgs(g *generator.Generator, model *struc.Model, constructorName string, returnVal, exportMethods bool, nolint bool) (string, string, error) {
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

	params := typeparams.New(model.Typ.TypeParams())
	typeParams := params.IdentString(g.OutPkgPath)
	typeParamsDecl := params.DeclarationString(g.OutPkgPath)
	uniqueNames := unique.NewNamesWith(unique.DistinctBySuffix("_"))
	seq.ForEach(params.Names(g.OutPkgPath), uniqueNames.Add)

	name, body := New(constructorName, model.TypeName(), typeParamsDecl, typeParams, uniqueNames.Get("r"), returnVal, exportMethods, nolint, args, initPart, nil)
	return name, body, nil
}
