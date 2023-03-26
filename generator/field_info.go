package generator

import (
	"strings"

	"github.com/m4gshm/fieldr/struc"
)

type FieldInfo struct {
	Name string
	Type struc.FieldType
}

func FiledPathAndAccessCheckCondition(receiverVar string, isReceiverReference, useConditinonReceiver bool, fieldPath []FieldInfo) (string, string, []string) {
	nilReceiver := "r"
	conditions := []string{}
	shortConditionPath := ""
	if isReceiverReference {
		if useConditinonReceiver {
			conditions = append(conditions, nilReceiver+":="+receiverVar+";"+nilReceiver+"!=nil")
			shortConditionPath = nilReceiver
		} else {
			conditions = append(conditions, receiverVar+" != nil")
		}
	}
	fullFieldPath := receiverVar
	for _, p := range fieldPath {
		if len(fullFieldPath) > 0 {
			fullFieldPath += "."
		}
		fullFieldPath += p.Name
		if useConditinonReceiver {
			shortConditionPath = ifElse(len(shortConditionPath) > 0, shortConditionPath+"."+p.Name, fullFieldPath)
		}
		if p.Type.RefCount > 0 {
			if useConditinonReceiver {
				conditions = append(conditions, nilReceiver+":="+shortConditionPath+";"+nilReceiver+" != nil")
				shortConditionPath = nilReceiver
			} else {
				conditions = append(conditions, fullFieldPath+" != nil")
			}
			for ri := 1; ri < p.Type.RefCount; ri++ {
				if useConditinonReceiver {
					shortConditionPathRef := "*" + shortConditionPath + ""
					conditions = append(conditions, nilReceiver+":="+shortConditionPathRef+";"+nilReceiver+" != nil")
					// shortConditionPath = "(" + shortConditionPath + ")"
				} else {
					fullFieldPath = "*(" + fullFieldPath + ")"
					conditions = append(conditions, fullFieldPath+" != nil")
					fullFieldPath = "(" + fullFieldPath + ")"
				}
			}
		}
	}
	if !useConditinonReceiver && len(conditions) > 0 {
		conditions = []string{strings.Join(conditions, " && ")}
	}
	return fullFieldPath, shortConditionPath, conditions
}
