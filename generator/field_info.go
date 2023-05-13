package generator

import (
	"strings"
	"unicode"

	"github.com/m4gshm/fieldr/struc"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/op/get"
	"github.com/m4gshm/gollections/slice"
)

type FieldInfo struct {
	Name string
	Type struc.FieldType
}

func FiledPathAndAccessCheckCondition(receiverVar string, isReceiverReference, redeclareReceiver bool, fieldParts []FieldInfo) (string, string, []string) {
	conditions := []string{}
	shortConditionPath := ""
	if isReceiverReference {
		newReceiver := PathToShortVarName(receiverVar)
		conditions = slice.Of(get.If(redeclareReceiver, sum.Of(newReceiver, ":=", receiverVar, ";", newReceiver, "!=nil")).ElseGet(sum.Of(newReceiver, "!=nil")))
		shortConditionPath = newReceiver
	}
	fieldPath := ""
	for _, part := range fieldParts {
		fieldPath += op.IfElse(len(fieldPath) > 0, ".", "") + part.Name

		shortConditionPath = get.If(len(shortConditionPath) > 0, sum.Of(shortConditionPath, ".", part.Name)).ElseGet(func() string {
			return receiverVar + get.If(len(fieldPath) > 0, sum.Of(".", fieldPath)).Else("")
		})

		if part.Type.RefCount > 0 {
			newReceiver := PathToShortVarName(part.Name)
			conditions = append(conditions, newReceiver+":="+shortConditionPath+";"+newReceiver+" != nil")
			shortConditionPath = newReceiver

			for ri := 1; ri < part.Type.RefCount; ri++ {
				shortConditionPathRef := "*" + shortConditionPath
				newReceiver := PathToShortVarName(shortConditionPathRef)
				conditions = append(conditions, newReceiver+":="+shortConditionPathRef+";"+newReceiver+" != nil")
				shortConditionPath = newReceiver
			}
		}
	}
	return fieldPath, shortConditionPath, conditions
}

func PathToVarName(fieldPath string) string {
	return strings.NewReplacer(".", "_", "*", "_").Replace(fieldPath)
}

func PathToShortVarName(fieldPath string) string {
	if len(fieldPath) == 0 {
		return "r"
	} else if parts := strings.Split(fieldPath, "."); len(parts) > 1 {
		return strings.Join(slice.Convert(parts, PathToShortVarName), "_")
	} else if parts := strings.Split(fieldPath, "_"); len(parts) > 1 {
		return strings.Join(slice.Convert(parts, PathToShortVarName), "_")
	}

	body := []rune{}
	pref := []rune{}
	hasLetter := false
	for _, r := range fieldPath {
		if unicode.IsLower(r) && !hasLetter {
			body = append(body, r)
			hasLetter = true
		} else if unicode.IsUpper(r) {
			hasLetter = true
			body = append(body, unicode.ToLower(r))
		} else if unicode.IsDigit(r) {
			body = append(body, r)
		} else if r == '_' {
			body = append(body, r)
		} else if r == '*' {
			pref = append(pref, '_')
		} else if r == '.' {
			body = append(body, '_')
		}
	}
	if len(body) == 0 {
		body = []rune{'r'}
	}
	path := string(pref) + string(body)
	return path
}

func TypeReceiverVar(typeName string) string {
	if parts := strings.Split(typeName, "."); len(parts) > 1 {
		if converted := slice.Convert(parts, TypeReceiverVar); len(converted) > 1 {
			if len(converted[1]) > 0 {
				return converted[1]
			} else if len(converted[0]) > 0 {
				return converted[0]
			}
		}
	} else if f, ok := slice.First([]rune(typeName), unicode.IsLetter); ok {
		return string(unicode.ToLower(f))
	}
	return "r"
}
