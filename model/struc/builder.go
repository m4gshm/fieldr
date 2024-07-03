package struc

import (
	"fmt"
	"go/types"
	"reflect"

	"github.com/m4gshm/gollections/convert/as"
	"github.com/m4gshm/gollections/map_"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/util"
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
	fieldTagValues := slice.Map(fieldTagNames, as.Is[TagName], map_.Getter(tagValues))
	if len(fieldTagValues) > 0 {
		b.model.FieldsTagValue[fldName] = fieldTagValues
	}
}

func (b *structModelBuilder) populateByStruct(typ *types.Struct) error {
	numFields := typ.NumFields()
	for i := 0; i < numFields; i++ {
		fieldVar := typ.Field(i)
		if !fieldVar.IsField() {
			return fmt.Errorf("unexpected struct element, must be field, value %v, type %v", fieldVar, reflect.TypeOf(fieldVar))
		}
		fldName := fieldVar.Name()
		if _, ok := b.model.FieldsType[fldName]; ok {
			logger.Infof("duplicated field '%s'", fldName)
			continue
		}
		b.model.FieldNames = append(b.model.FieldNames, fldName)

		tagValues, fieldTagNames := parseTagValues(typ.Tag(i))
		b.populateFields(fldName, fieldTagNames, tagValues)
		for _, fieldTagName := range fieldTagNames {
			b.populateTags(fldName, fieldTagName, tagValues[fieldTagName])
		}
		fieldType := fieldVar.Type()
		fieldTypeName := util.TypeString(fieldType, b.outPkgPath)
		ref := 0
		var fieldModel *Model
		if structType, p := util.GetStructTypeNamed(fieldType); structType != nil {
			typeName := structType.Obj().Name()
			ref = p
			fieldTypeName = typeName
			if b.deep {
				if fmodel, ok := b.loopControl[structType]; ok {
					logger.Debugf("found handled type %v", typeName)
					fieldModel = fmodel
				} else if model, err := newBuilder(b.outPkgPath, b.loopControl).newModel(structType); err != nil {
					return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
				} else {
					fieldModel = model
				}
			}
		}
		b.model.FieldsType[fldName] = FieldType{
			Embedded: fieldVar.Embedded(), RefCount: ref, Name: fieldTypeName,
			FullName: util.TypeString(fieldType, b.outPkgPath),
			Type:     fieldType, Model: fieldModel,
		}
	}
	return nil
}

func (b *structModelBuilder) newModel(typ *types.Named) (*Model, error) {
	obj := typ.Obj()
	typName := obj.Name()
	if _, ok := b.loopControl[typ]; ok {
		return nil, fmt.Errorf("already handled type %v", typName)
	}
	typStruct, rc := util.GetTypeStruct(typ)
	if typStruct == nil {
		return nil, fmt.Errorf("'%s' is not a struct type", typName)
	}

	model := &Model{
		Typ:            typ,
		typeName:       typName,
		pkg:            obj.Pkg(),
		OutPkgPath:     b.outPkgPath,
		FieldsTagValue: map[FieldName]map[TagName]TagValue{},
		TagsFieldValue: map[TagName]map[FieldName]TagValue{},
		FieldNames:     []FieldName{},
		FieldsType:     map[FieldName]FieldType{},
		RefCount:       rc,
	}
	b.loopControl[typ] = model
	b.model = model

	if err := b.populateByStruct(typStruct); err != nil {
		return nil, err
	}
	return b.model, nil
}
