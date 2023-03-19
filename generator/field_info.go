package generator

import "github.com/m4gshm/fieldr/struc"

type FieldInfo struct {
	Name string
	Type struc.FieldType
}

func FiledPathAndAccessCheckCondition(receiverVar string, isReceiverReference bool, fieldPathInfo []FieldInfo) (string, string) {
	condition := ""
	if isReceiverReference {
		condition += receiverVar + " != nil"
	}
	fieldPath := ""
	fullFieldPath := receiverVar + "."
	for _, p := range fieldPathInfo {
		if len(fieldPath) > 0 {
			fieldPath += "."
			fullFieldPath += "."
		}
		fieldPath += p.Name
		fullFieldPath += p.Name
		if p.Type.RefCount > 0 {
			if len(condition) > 0 {
				condition += " && "
			}
			condition += fullFieldPath + " != nil"
			for ri := 1; ri < p.Type.RefCount; ri++ {
				condition += " && "
				fullFieldPath = "*(" + fullFieldPath + ")"
				condition += fullFieldPath + " != nil"
				fullFieldPath = "(" + fullFieldPath + ")"
			}
		}
	}
	return fullFieldPath, condition
}
