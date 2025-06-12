package generator

import (
	"strings"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op/delay/sum"
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
