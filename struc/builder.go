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

func newBuilder(outPkgPath string, loopControl handledStructs) (*structModelBuilder, error) {
	return &structModelBuilder{
		deep:        true,
		outPkgPath:  outPkgPath,
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
						} else if nestedBuilder, err := newBuilder(b.outPkgPath, b.loopControl); err != nil {
							return err
						} else if model, err = nestedBuilder.newModel(pkg, structType); err != nil {
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

func (b *structModelBuilder) newModel(typPack *types.Package, typ *types.Named) (*Model, error) {
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
		Package:        Package{Name: typPack.Name(), Path: typPack.Path()},
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

func GetTypeNamed(typ types.Type) (*types.Named, int, error) {
	switch ftt := typ.(type) {
	case *types.Named:
		return ftt, 0, nil
	case *types.Pointer:
		t, p, err := GetTypeNamed(ftt.Elem())
		if err != nil {
			return nil, 0, err
		}
		return t, p + 1, nil
	default:
		return nil, 0, nil
	}
}

func GetStructTypeNamed(typ types.Type) (*types.Named, int, error) {
	if ftt, p, err := GetTypeNamed(typ); err != nil {
		return nil, 0, err
	} else if ftt != nil {
		und := ftt.Underlying()
		if _, ok := und.(*types.Struct); ok {
			return ftt, p, nil
		} else if sund, sp, err := GetStructTypeNamed(und); err != nil {
			return nil, sp + p, err
		} else if sund != nil {
			return ftt, sp + p, nil
		}
	}
	return nil, 0, nil
	// switch ftt := typ.(type) {
	// case *types.Named:
	// 	und := ftt.Underlying()
	// 	if _, ok := und.(*types.Struct); ok {
	// 		return ftt, 0, nil
	// 	} else if sund, p, err := GetStructTypeNamed(und); err != nil {
	// 		return nil, 0, err
	// 	} else if sund != nil {
	// 		return ftt, p, nil
	// 	}
	// 	return nil, 0, nil
	// case *types.Pointer:
	// 	t, p, err := GetStructTypeNamed(ftt.Elem())
	// 	if err != nil {
	// 		return nil, 0, err
	// 	}
	// 	return t, p + 1, nil
	// default:
	// 	return nil, 0, nil
	// }
}

func GetStructType(t types.Type) (*types.Struct, int, error) {
	switch tt := t.(type) {
	case *types.Struct:
		return tt, 0, nil
	case *types.Pointer:
		s, pc, err := GetStructType(tt.Elem())
		if err != nil {
			return nil, 0, err
		}
		return s, pc + 1, nil
	case *types.Named:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0, nil
		}
		return GetStructType(underlying)
	case types.Type:
		underlying := tt.Underlying()
		if underlying == t {
			return nil, 0, nil
		}
		return GetStructType(underlying)
	default:
		return nil, 0, nil
	}
}
