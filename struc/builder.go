package struc

import (
	"fmt"
	"go/types"
	"reflect"

	"github.com/m4gshm/fieldr/logger"
)

type handledStructs = map[types.Type]*HierarchicalModel

type structModelBuilder struct {
	model       *HierarchicalModel
	deep        bool
	pkg         *types.Package
	loopControl handledStructs
}

func newBuilder(pkg *types.Package, typ types.Type, typeName string, filePath string, loopControl handledStructs) (*structModelBuilder, error) {
	if _, ok := loopControl[typ]; ok {
		return nil, fmt.Errorf("already handled type %v", typeName)
	}
	model := &HierarchicalModel{
		Model: Model{
			TypeName:       typeName,
			FilePath:       filePath,
			PackageName:    pkg.Name(),
			PackagePath:    pkg.Path(),
			FieldsTagValue: map[FieldName]map[TagName]TagValue{},
			TagsFieldValue: map[TagName]map[FieldName]TagValue{},
			FieldNames:     []FieldName{},
			FieldsType:     map[FieldName]FieldType{},
			TagNames:       []TagName{}},
		Nested: map[FieldName]*HierarchicalModel{},
	}
	loopControl[typ] = model
	return &structModelBuilder{
		model:       model,
		deep:        true,
		pkg:         pkg,
		loopControl: loopControl,
	}, nil
}

func (b *structModelBuilder) populateTags(fieldName FieldName, tagName TagName, tagValue TagValue) {
	tagFields, tagFieldsOk := b.model.TagsFieldValue[tagName]
	if !tagFieldsOk {
		tagFields = make(map[FieldName]TagValue)
		b.model.TagsFieldValue[tagName] = tagFields
		b.model.TagNames = append(b.model.TagNames, tagName)
	}
	tagFields[fieldName] = tagValue
}

func (b *structModelBuilder) populateFields(fldName FieldName, fieldTagNames []TagName, tagValues map[TagName]TagValue) {
	fieldTagValues := newFieldTagValues(fieldTagNames, tagValues)
	if len(fieldTagValues) > 0 {
		b.model.FieldsTagValue[fldName] = fieldTagValues
	}
}

func (b *structModelBuilder) populateByStruct(typeStruct *types.Struct) error {
	numFields := typeStruct.NumFields()
	for i := 0; i < numFields; i++ {
		fieldVar := typeStruct.Field(i)
		fldName := fieldVar.Name()
		if fieldVar.IsField() {
			fieldType := fieldVar.Type()
			if fieldVar.Embedded() {
				if err := b.populateByType(fieldType); err != nil {
					return err
				}
			} else if _, ok := b.model.FieldsType[fldName]; ok {
				logger.Infof("duplicated field '%s'", fldName)
			} else {
				tag := typeStruct.Tag(i)

				b.model.FieldNames = append(b.model.FieldNames, fldName)

				tagValues, fieldTagNames := parseTagValues(tag)
				b.populateFields(fldName, fieldTagNames, tagValues)
				for _, fieldTagName := range fieldTagNames {
					b.populateTags(fldName, fieldTagName, tagValues[fieldTagName])
				}

				fieldTypeStr := fieldType.String()
				if fieldTypeNamed, err := getTypeNamed(fieldType); err != nil {
					return err
				} else if fieldTypeNamed != nil {
					var (
						obj      = fieldTypeNamed.Obj()
						pkg      = obj.Pkg()
						typeName = obj.Name()
					)
					if pkg == b.pkg {
						fieldTypeStr = typeName
					} else {
						fieldTypeStr = pkg.Name() + "." + typeName
					}
					if b.deep {
						if model, ok := b.loopControl[fieldTypeNamed]; ok {
							logger.Debugf("found handled type %v", typeName)
							b.model.Nested[fldName] = model
						} else if nestedBuilder, err := newBuilder(pkg, fieldTypeNamed, typeName, "", b.loopControl); err != nil {
							return err
						} else if err = nestedBuilder.populateByType(fieldTypeNamed); err != nil {
							return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
						} else {
							b.model.Nested[fldName] = nestedBuilder.getModel()
						}
					}
				}

				b.model.FieldsType[fldName] = FieldType(fieldTypeStr)
			}
		} else {
			return fmt.Errorf("unexpected struct element, must be field, value %v, type %v", fieldVar, reflect.TypeOf(fieldVar))
		}
	}
	return nil
}

func getTypeNamed(fieldType types.Type) (*types.Named, error) {
	switch ftt := fieldType.(type) {
	//case *types.Basic:
	//	return nil, nil
	case *types.Named:
		return ftt, nil
	case *types.Pointer:
		return getTypeNamed(ftt.Elem())
	default:
		return nil, nil
		//return nil, fmt.Errorf("unexpected field type %v, refl %v", fieldType, reflect.TypeOf(fieldType))
	}
}

func (b *structModelBuilder) populateByType(t types.Type) error {
	switch tt := t.(type) {
	case *types.Struct:
		return b.populateByStruct(tt)
	case types.Type:
		underlying := tt.Underlying()
		if underlying == t {
			return nil
		}
		return b.populateByType(underlying)
	default:
		return nil
	}
}

func (b *structModelBuilder) newModel(t types.Type) (*HierarchicalModel, error) {
	if err := b.populateByType(t); err != nil {
		return nil, err
	}
	return b.getModel(), nil
}

func (b *structModelBuilder) getModel() *HierarchicalModel {
	return b.model
}
