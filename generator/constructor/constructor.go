package constructor

import (
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/predicate/always"
	"github.com/m4gshm/gollections/seq"
	"github.com/m4gshm/gollections/seq2"

	"github.com/m4gshm/fieldr/generator"
	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/typeargs"
	"github.com/m4gshm/fieldr/typeparams"
	"github.com/m4gshm/fieldr/unique"
)

func New(
	name, typeName, typeParamsDecl, typeParams, receiver string,
	returnVal, exportMethods, nolint bool,
	arguments, createInstance string, init func(receiver string) string,
) (string, string) {
	constructorName := generator.IdentName(get.If(name == generator.Autoname, sum.Of("New", typeName)).Else(name), exportMethods)

	initPart := ""
	if init != nil {
		initPart = init(receiver)
		if len(initPart) > 0 {
			initPart += "\n"
		}
	}

	body := "func " + constructorName + typeParamsDecl + "(" + op.IfElse(len(arguments) > 0, "\n", "") + arguments + ") " + op.IfElse(returnVal, "", "*") +
		typeName + typeParams + " {" + generator.NoLint(nolint) + "\n"

	if len(initPart) > 0 {
		body += "r := " + createInstance + "\n"
		body += initPart
		body += "return " + receiver + "\n"
	} else {
		body += "return " + createInstance
	}
	body += "}\n"
	return constructorName, body
}

func FullArgs(g *generator.Generator, model *struc.Model, constructorName string, returnVal, exportMethods, nolint, flat bool) (string, string, error) {
	uniqueNames := unique.NewNamesWith(unique.DistinctBySuffix("_"))

	params := typeparams.New(model.Typ.TypeParams())
	typeParams, typeParamsDecl := params.IdentDeclStrings(g.OutPkgPath)
	params.Names(g.OutPkgPath).ForEach(uniqueNames.Add)

	typeName := model.TypeName()

	fields := model.FieldsNameAndType
	args, createInstance, err := GenerateConstructorArgs(g, uniqueNames, typeName, typeParams, fields, returnVal, flat, always.True)
	if err != nil {
		return "", "", err
	}
	name, body := New(constructorName, typeName, typeParamsDecl, typeParams, uniqueNames.Get("r"), returnVal, exportMethods, nolint, args, createInstance, nil)
	return name, body, nil
}

var noFields = seq2.Of[struc.FieldName, struc.FieldType]()

func GenerateConstructorArgs(
	g *generator.Generator, uniqueNames *unique.Names, typeName string, typeParams string,
	fields seq.Seq2[struc.FieldName, struc.FieldType],
	returnVal, flat bool, isInclude func(struc.FieldName) bool,
) (string, string, error) {
	var args, initInstace string
	for fieldName, fieldType := range op.IfElse(fields != nil, fields, noFields) {
		deepRef := fieldType.RefDeep > 1
		if !deepRef && fieldType.Embedded && flat {
			model := fieldType.Model
			typ := model.Typ
			typeParams := typeargs.New(typ.TypeArgs()).IdentString(g.OutPkgPath)
			val := fieldType.RefDeep == 0
			eargs, eCreateInstance, err := GenerateConstructorArgs(g, uniqueNames, model.TypeName(), typeParams, model.FieldsNameAndType, val, flat, isInclude)
			if err != nil {
				return "", "", err
			}
			args += eargs
			initInstace += fieldName + ":" + eCreateInstance + op.IfElse(len(eCreateInstance) > 0, ",\n", "")
		} else if isInclude(fieldName) {
			fullFieldType, err := g.GetFullFieldTypeName(fieldType, false)
			if err != nil {
				return "", "", err
			}
			argName := uniqueNames.Get(fieldName)
			args += argName + " " + fullFieldType + ",\n"
			initInstace += fieldName + ":" + argName + ",\n"
		}
	}
	createInstance := op.IfElse(returnVal, "", "&") + typeName + typeParams + "{ " + op.IfElse(len(initInstace) > 0, "\n", "") + initInstace + " }"
	return args, createInstance, nil
}
