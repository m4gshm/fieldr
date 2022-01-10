package struc

import (
	"fmt"
	"go/types"
	"reflect"

	"github.com/m4gshm/fieldr/logger"
)

type handledStructs = map[types.Type]*Model

type structModelBuilder struct {
	model       *Model
	deep        bool
	rootPack    *types.Package
	loopControl handledStructs
}

func newBuilder(rootPack, modelPack *types.Package, typ types.Type, typeName string, filePath string, loopControl handledStructs) (*structModelBuilder, error) {
	if _, ok := loopControl[typ]; ok {
		return nil, fmt.Errorf("already handled type %v", typeName)
	}
	model := &Model{
		TypeName:       typeName,
		FilePath:       filePath,
		Package:        Package{Name: modelPack.Name(), Path: modelPack.Path()},
		FieldsTagValue: map[FieldName]map[TagName]TagValue{},
		TagsFieldValue: map[TagName]map[FieldName]TagValue{},
		FieldNames:     []FieldName{},
		FieldsType:     map[FieldName]FieldType{},
	}
	loopControl[typ] = model
	return &structModelBuilder{
		model:       model,
		deep:        true,
		rootPack:    rootPack,
		loopControl: loopControl,
	}, nil
}

func (b *structModelBuilder) populateTags(fieldName FieldName, tagName TagName, tagValue TagValue) {
	tagFields, tagFieldsOk := b.model.TagsFieldValue[tagName]
	if !tagFieldsOk {
		tagFields = make(map[FieldName]TagValue)
		b.model.TagsFieldValue[tagName] = tagFields
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
			embedded := fieldVar.Embedded()
			var fieldModel *Model
			if _, ok := b.model.FieldsType[fldName]; ok {
				logger.Infof("duplicated field '%s'", fldName)
			} else {
				tag := typeStruct.Tag(i)

				b.model.FieldNames = append(b.model.FieldNames, fldName)

				tagValues, fieldTagNames := parseTagValues(tag)
				b.populateFields(fldName, fieldTagNames, tagValues)
				for _, fieldTagName := range fieldTagNames {
					b.populateTags(fldName, fieldTagName, tagValues[fieldTagName])
				}
				fieldTypeName := TypeString(fieldType, b.rootPack.Name())
				ref := false
				if structType, p, err := GetStructTypeName(fieldType); err != nil {
					return err
				} else if structType != nil {
					ref = p
					var (
						obj      = structType.Obj()
						pkg      = obj.Pkg()
						typeName = obj.Name()
					)
					fieldTypeName = typeName
					if b.deep {
						if model, ok := b.loopControl[structType]; ok {
							logger.Debugf("found handled type %v", typeName)
							fieldModel = model
						} else if nestedBuilder, err := newBuilder(b.rootPack, pkg, structType, typeName, "", b.loopControl); err != nil {
							return err
						} else if err = nestedBuilder.populateByType(structType); err != nil {
							return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
						} else {
							fieldModel = nestedBuilder.getModel()
						}
					}
				}

				ft := FieldType{
					Embedded: embedded, Ref: ref, Name: fieldTypeName,
					FullName: TypeString(fieldType, b.rootPack.Name()),
					Type:     fieldType, Model: fieldModel,
				}
				b.model.FieldsType[fldName] = ft
			}
		} else {
			return fmt.Errorf("unexpected struct element, must be field, value %v, type %v", fieldVar, reflect.TypeOf(fieldVar))
		}
	}
	return nil
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

func (b *structModelBuilder) newModel(t types.Type) (*Model, error) {
	if err := b.populateByType(t); err != nil {
		return nil, err
	}
	return b.getModel(), nil
}

func (b *structModelBuilder) getModel() *Model {
	return b.model
}

func GetStructTypeName(fieldType types.Type) (*types.Named, bool, error) {
	switch ftt := fieldType.(type) {
	case *types.Named:
		und := ftt.Underlying()
		if _, ok := und.(*types.Struct); ok {
			return ftt, false, nil
		} else if sund, _, err := GetStructTypeName(und); err != nil {
			return nil, false, err
		} else if sund != nil {
			return ftt, false, nil
		}
		return nil, false, nil
	case *types.Pointer:
		t, _, err := GetStructTypeName(ftt.Elem())
		if err != nil {
			return nil, true, err
		}
		return t, true, nil
	default:
		return nil, false, nil
	}
}
