package command

import "github.com/m4gshm/fieldr/struc"

func toFlatModel(hierarchicalModel *struc.HierarchicalModel, flatFields []string) *struc.Model {
	flatFieldSet := make(map[struc.FieldName]struct{})

	for _, flatField := range flatFields {
		flatFieldSet[flatField] = struct{}{}
	}

	existsFlatFields := make(map[struc.FieldName]interface{})
	for _, fieldName := range hierarchicalModel.FieldNames {
		if _, nested := flatFieldSet[fieldName]; nested {
			existsFlatFields[fieldName] = nil
		}
	}

	var model *struc.Model
	if len(existsFlatFields) > 0 {
		//make flat model
		var (
			flatFieldNames     []struc.FieldName
			flatFieldsType     = map[struc.FieldName]struc.FieldType{}
			flatFieldsTagValue = map[struc.FieldName]map[struc.TagName]struc.TagValue{}
		)
		for _, fieldName := range hierarchicalModel.FieldNames {
			if _, ok := existsFlatFields[fieldName]; ok {
				if nestedHierarchicalModel := hierarchicalModel.Nested[fieldName]; nestedHierarchicalModel != nil {
					nestedModel := nestedHierarchicalModel.Model
					for _, nestedFieldName := range nestedModel.FieldNames {
						nestedFieldRef := struc.GetFieldRef(fieldName, nestedFieldName)

						flatFieldsType[nestedFieldRef] = nestedHierarchicalModel.FieldsType[nestedFieldName]
						flatFieldsTagValue[nestedFieldRef] = nestedHierarchicalModel.FieldsTagValue[nestedFieldName]

						flatFieldNames = append(flatFieldNames, nestedFieldRef)
					}
				} else {
					flatFieldNames = append(flatFieldNames, fieldName)
				}
			} else {
				flatFieldNames = append(flatFieldNames, fieldName)
			}
			flatFieldsType[fieldName] = hierarchicalModel.FieldsType[fieldName]
			flatFieldsTagValue[fieldName] = hierarchicalModel.FieldsTagValue[fieldName]
		}

		tagsFieldValue := map[struc.TagName]map[struc.FieldName]struc.TagValue{}
		for fieldName, tagNameValues := range flatFieldsTagValue {
			for tagName, tagValue := range tagNameValues {
				fieldTagValues, ok := tagsFieldValue[tagName]
				if !ok {
					fieldTagValues = map[struc.FieldName]struc.TagValue{}
				}
				fieldTagValues[fieldName] = tagValue
				tagsFieldValue[tagName] = fieldTagValues
			}
		}

		model = &struc.Model{
			TypeName:          hierarchicalModel.TypeName,
			PackageName:       hierarchicalModel.PackageName,
			PackagePath:       hierarchicalModel.PackagePath,
			FilePath:          hierarchicalModel.FilePath,
			FieldsTagValue:    flatFieldsTagValue,
			TagsFieldValue:    tagsFieldValue,
			FieldNames:        flatFieldNames,
			FieldsType:        flatFieldsType,
			TagNames:          hierarchicalModel.TagNames,
			Constants:         hierarchicalModel.Constants,
			ConstantTemplates: hierarchicalModel.ConstantTemplates,
		}
	} else {
		model = &hierarchicalModel.Model
	}

	return model
}
