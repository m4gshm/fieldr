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
	outPkgPath  string
	loopControl handledStructs
}

func newBuilder(outPkgPath string, loopControl handledStructs) *structModelBuilder {
	return &structModelBuilder{
		deep:        true,
		outPkgPath:  outPkgPath,
		loopControl: loopControl,
	}
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

func (b *structModelBuilder) populateByStruct(typ *types.Struct) error {
	numFields := typ.NumFields()
	for i := 0; i < numFields; i++ {
		fieldVar := typ.Field(i)
		fldName := fieldVar.Name()
		if fieldVar.IsField() {
			fieldType := fieldVar.Type()
			embedded := fieldVar.Embedded()
			var fieldModel *Model
			if _, ok := b.model.FieldsType[fldName]; ok {
				logger.Infof("duplicated field '%s'", fldName)
			} else {
				tag := typ.Tag(i)

				b.model.FieldNames = append(b.model.FieldNames, fldName)

				tagValues, fieldTagNames := parseTagValues(tag)
				b.populateFields(fldName, fieldTagNames, tagValues)
				for _, fieldTagName := range fieldTagNames {
					b.populateTags(fldName, fieldTagName, tagValues[fieldTagName])
				}
				fieldTypeName := TypeString(fieldType, b.outPkgPath)
				ref := 0
				if structType, p, err := GetStructTypeNamed(fieldType); err != nil {
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
						} else if model, err = newBuilder(b.outPkgPath, b.loopControl).newModel(Package{Name: pkg.Name(), Path: pkg.Path()}, structType); err != nil {
							return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
						} else {
							fieldModel = model
						}
					}
				}
				ft := FieldType{
					Embedded: embedded, RefCount: ref, Name: fieldTypeName,
					FullName: TypeString(fieldType, b.outPkgPath),
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

func (b *structModelBuilder) newModel(pkg Package, typ *types.Named) (*Model, error) {
	typName := typ.Obj().Name()
	if _, ok := b.loopControl[typ]; ok {
		return nil, fmt.Errorf("already handled type %v", typName)
	}
	st, rc, err := GetStructType(typ)
	if err != nil {
		return nil, err
	}
	model := &Model{
		Typ:            typ,
		TypeName:       typName,
		Package:        pkg,
		OutPkgPath:     b.outPkgPath,
		FieldsTagValue: map[FieldName]map[TagName]TagValue{},
		TagsFieldValue: map[TagName]map[FieldName]TagValue{},
		FieldNames:     []FieldName{},
		FieldsType:     map[FieldName]FieldType{},
		RefCount:       rc,
	}
	b.loopControl[typ] = model
	b.model = model

	if err := b.populateByStruct(st); err != nil {
		return nil, err
	}
	return b.model, nil
}
