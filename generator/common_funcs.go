package generator

import (
	"go/types"

	"github.com/m4gshm/fieldr/model/util"
	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/loop"
	"github.com/m4gshm/gollections/loop/convert"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/op/string_"
)

func MethodName(typ, fun string) string { return typ + "." + fun }

func FuncBodyNoArg(name string, returnType string, nolint bool, content string) string {
	return "func " + name + "()" + returnType + " {" + NoLint(nolint) + "\n" + content + "\n}\n"
}

func MethodBody(name string, isFunc bool, methodReceiverVar, methodReceiverType, returnType string, nolint bool, content string) string {
	return "func " + get.If(isFunc,
		sum.Of(name, "(", methodReceiverVar, " ", methodReceiverType, ")"),
	).ElseGet(
		sum.Of("(", methodReceiverVar, " ", methodReceiverType, ") ", name, "()"),
	) + " " + returnType + " {" + NoLint(nolint) + "\n" + content + "\n}\n"
}

func TypeParamsString(tparams *types.TypeParamList, basePkgPath string) string {
	return string_.WrapNonEmpty("[", loop.Reduce(convert.FromIndexed(tparams.Len(), tparams.At, func(elem *types.TypeParam) string {
		return use.If(elem == nil, "/*error: nil type parameter*/").ElseGet(
			func() string { return util.TypeString(elem, basePkgPath) })
	}), join.NonEmpty(", ")), "]")
}

func TypeParamsDeclarationString(list *types.TypeParamList, basePkgPath string) string {
	var (
		prevElem types.Type
		noFirst  = false
	)
	return string_.WrapNonEmpty("[", loop.Reduce(convert.FromIndexed(list.Len(), list.At, func(elem *types.TypeParam) string {
		s := use.If(elem == nil, "/*error: nil type parameter*/").ElseGet(func() string {
			constraint := elem.Constraint()
			s := use.If(!noFirst, "").IfGet(constraint != prevElem, sum.Of(" ", util.TypeString(prevElem, basePkgPath), ",")).Else(",")
			prevElem = constraint
			return s + util.TypeString(elem, basePkgPath)
		})
		noFirst = true
		return s
	}), op.Sum)+get.If(prevElem != nil, func() string { return " " + util.TypeString(prevElem, basePkgPath) }).Else(""), "]")
}
