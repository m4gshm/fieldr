package generator

import (
	"strings"
	"unicode"

	"github.com/m4gshm/gollections/expr/get"
	"github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/op/delay/sum"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/model/struc"
	"github.com/m4gshm/fieldr/unique"
)

type FieldInfo struct {
	Name string
	Type struc.FieldType
}

func FiledPathAndAccessCheckCondition(receiverVar string, isReceiverReference, redeclareReceiver bool, fieldParts []FieldInfo, uniqueNames *unique.Names) (string, string, []string) {
	conditions := []string{}
	if isReceiverReference {
		newReceiver := receiverVar
		receiverCondition := get.If(redeclareReceiver, sum.Of(newReceiver, ":=", receiverVar, ";", newReceiver, "!=nil")).ElseGet(sum.Of(newReceiver, "!=nil"))
		conditions = append(conditions, receiverCondition)
		receiverVar = newReceiver
	}
	accessInfo := GetFieldConditionalPartsAccessInfo(receiverVar, fieldParts, uniqueNames)
	for _, cp := range accessInfo.AccessPathParts {
		conditions = append(conditions, cp.ShortVar+":="+cp.FieldPath+";"+cp.ShortVar+"!=nil")
	}
	return accessInfo.FieldPath, accessInfo.ShortVar, conditions
}

type FieldConditionalPartsAccessInfo struct {
	FieldPath       string
	ShortVar        string
	AccessPathParts []AccessPathPart
}

type AccessPathPart struct {
	FieldPath string
	ShortVar  string
	Type      *struc.FieldType
}

func GetFieldConditionalPartsAccessInfo(receiverVar string, fieldParts []FieldInfo, uniqueNames *unique.Names) FieldConditionalPartsAccessInfo {
	conditionParts := []AccessPathPart{}
	shortVar := receiverVar
	fieldPath := receiverVar
	for _, part := range fieldParts {
		fieldPath += op.IfElse(len(fieldPath) > 0, ".", "") + part.Name
		partShortPath := get.If(len(shortVar) > 0, sum.Of(shortVar, ".", part.Name)).Else("")
		if partType := part.Type; partType.RefDeep == 0 {
			shortVar = partShortPath
		} else {
			parthShortVar := uniqueNames.Get(PathToShortVarName(part.Name))
			conditionParts = append(conditionParts, AccessPathPart{ShortVar: parthShortVar, FieldPath: partShortPath, Type: &partType})
			for ri := 1; ri < partType.RefDeep; ri++ {
				parthShortVarRef := "*" + parthShortVar
				newReceiver := uniqueNames.Get(PathToShortVarName(parthShortVarRef))

				ut := partType.Type.Underlying()
				ctyp := struc.NewFieldType(partType.Embedded, partType.RefDeep-1, partType.Name, ut, partType.Model)

				conditionParts = append(conditionParts, AccessPathPart{ShortVar: newReceiver, FieldPath: parthShortVarRef, Type: &ctyp})
				parthShortVar = newReceiver
			}
			shortVar = parthShortVar
		}
	}
	return FieldConditionalPartsAccessInfo{
		FieldPath:       fieldPath,
		ShortVar:        shortVar,
		AccessPathParts: conditionParts,
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
