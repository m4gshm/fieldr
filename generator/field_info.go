package generator

import (
	"go/types"
	"strings"
	"unicode"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/struc"
)

type FieldInfo struct {
	Name string
	Type struc.FieldType
}

func FiledPathAndAccessCheckCondition(receiverVar string, isReceiverReference, redeclareReceiver bool, fieldParts []FieldInfo) (string, string, []string) {
	conditions := []string{}
	if isReceiverReference {
		newReceiver := receiverVar
		receiverCondition := get.If(redeclareReceiver, sum.Of(newReceiver, ":=", receiverVar, ";", newReceiver, "!=nil")).ElseGet(sum.Of(newReceiver, "!=nil"))
		conditions = append(conditions, receiverCondition)
		receiverVar = newReceiver
	}

	i := GetFieldConditionalPartsAccessInfo(receiverVar, true, fieldParts)

	// for _, cp := range i.ConditionParts {
	// 	cp.ShortVar
	// }

	conditions = append(conditions, i.Conditions...)
	return i.FieldPath, i.ShortVar, conditions
}

type FieldConditionalPartsAccessInfo struct {
	FieldPath      string
	ShortVar       string
	ConditionParts []ConditionPart
	Conditions     []string
}

type ConditionPart struct {
	FieldPath string
	ShortVar  string
	Type      *struc.FieldType
}

func GetFieldConditionalPartsAccessInfo(receiverVar string, checkNotNil bool, fieldParts []FieldInfo) FieldConditionalPartsAccessInfo {
	conditionParts := []ConditionPart{}
	conditions := []string{}
	shortPath := receiverVar
	fullPath := receiverVar
	uniqueVars := NewUniqueShortVarGenerator(receiverVar)
	condition := op.IfElse(checkNotNil, "!=nil", "==nil")
	for _, part := range fieldParts {
		fullPath += op.IfElse(len(fullPath) > 0, ".", "") + part.Name
		partShortPath := get.If(len(shortPath) > 0, sum.Of(shortPath, ".", part.Name)).Else("")
		partType := part.Type
		if partType.RefDeep == 0 {
			shortPath = partShortPath
		} else {
			parthShortVar := uniqueVars.Get(PathToShortVarName(part.Name))
			conditionParts = append(conditionParts, ConditionPart{ShortVar: parthShortVar, FieldPath: partShortPath, Type: &partType})
			conditions = append(conditions, parthShortVar+":="+partShortPath+";"+parthShortVar+condition)
			for ri := 1; ri < partType.RefDeep; ri++ {
				parthShortVarRef := "*" + parthShortVar
				newReceiver := uniqueVars.Get(PathToShortVarName(parthShortVarRef))

				pt := partType.Type.(*types.Pointer)
				ut := pt.Underlying()
				ctyp := struc.NewFieldType(partType.Embedded, partType.RefDeep-1, partType.Name, ut, partType.Model)

				conditionParts = append(conditionParts, ConditionPart{ShortVar: newReceiver, FieldPath: parthShortVarRef, Type: &ctyp})
				conditions = append(conditions, newReceiver+":="+parthShortVarRef+";"+newReceiver+condition)
				parthShortVar = newReceiver
			}
			shortPath = parthShortVar
		}
	}
	return FieldConditionalPartsAccessInfo{
		FieldPath:      fullPath,
		ShortVar:       shortPath,
		Conditions:     conditions,
		ConditionParts: conditionParts,
	}
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
