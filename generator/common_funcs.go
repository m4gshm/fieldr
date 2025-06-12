package generator

import (
	"go/types"
	"strings"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/expr/use"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/string_/join"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/op/string_"
	"github.com/m4gshm/gollections/seq"

	"github.com/m4gshm/fieldr/model/util"
)

func MethodName(typ, fun string) string { return typ + "." + fun }

func FuncBodyNoArg(name string, returnType string, nolint bool, content string) string {
	return FuncBodyWithArgs(name, nil, returnType, nolint, content)
}

func FuncBodyWithArgs(name string, args []string, returnType string, nolint bool, content string) string {
	return "func " + name + "(" + strings.Join(args, ", ") + ")" + returnType + " {" + NoLint(nolint) + "\n" + content + "\n}\n"
}

func MethodBody(name string, isFunc bool, methodReceiverVar, methodReceiverType, returnType string, nolint bool, content string) string {
	return "func " + get.If(isFunc,
		sum.Of(name, "(", methodReceiverVar, " ", methodReceiverType, ")"),
	).ElseGet(
		sum.Of("(", methodReceiverVar, " ", methodReceiverType, ") ", name, "()"),
	) + " " + returnType + " {" + NoLint(nolint) + "\n" + content + "\n}\n"
}

func TypeParamsString(params seq.Seq[string]) string {
	return string_.WrapNonEmpty("[", seq.Reduce(params, join.NonEmpty(", ")), "]")
}

func TypeParamsSeq(tparams *types.TypeParamList, basePkgPath string) seq.Seq[string] {
	newVar := seq.Convert(seq.OfIndexed(tparams.Len(), tparams.At), func(elem *types.TypeParam) string {
		return use.If(elem == nil, "/*error: nil type parameter*/").ElseGet(
			func() string { return util.TypeString(elem, basePkgPath) })
	})
	return newVar
}

func TypeParamsDeclarationString(list *types.TypeParamList, basePkgPath string) string {
	var (
		prevElem types.Type
		noFirst  = false
	)
	return string_.WrapNonEmpty("[", seq.Reduce(seq.Convert(seq.OfIndexed(list.Len(), list.At), func(elem *types.TypeParam) string {
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
