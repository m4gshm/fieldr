package struc

import (
	"fmt"
	"go/types"
	"reflect"
)

type structModelBuilder struct {
	tags       map[TagName]map[FieldName]TagValue
	fields     map[FieldName]map[TagName]TagValue
	fieldNames []FieldName
	fieldsType map[FieldName]FieldType
	tagNames   []TagName

	includedTags map[TagName]interface{}
	deep         bool
	nested       map[FieldName]*StructModel
	pkg          *types.Package
}

func newBuilder(pkg *types.Package, includedTags map[TagName]interface{}, deep bool) *structModelBuilder {
	return &structModelBuilder{
		tags:         map[TagName]map[FieldName]TagValue{},
		fields:       map[FieldName]map[TagName]TagValue{},
		fieldNames:   []FieldName{},
		fieldsType:   map[FieldName]FieldType{},
		tagNames:     []TagName{},
		nested:       map[FieldName]*StructModel{},
		includedTags: includedTags,
		deep:         deep,
		pkg:          pkg,
	}
}

func (b *structModelBuilder) populateTags(fieldName FieldName, fieldTagName TagName, fieldTagValue TagValue) {
	tagFields, tagFieldsOk := b.tags[fieldTagName]
	if !tagFieldsOk {
		tagFields = make(map[FieldName]TagValue)
		b.tags[fieldTagName] = tagFields
		b.tagNames = append(b.tagNames, fieldTagName)
	}
	tagFields[fieldName] = fieldTagValue
}

func (b *structModelBuilder) populateFields(fldName FieldName, fieldTagNames []TagName, tagValues map[TagName]TagValue) {
	fieldTagValues := newFieldTagValues(fieldTagNames, tagValues)
	if len(fieldTagValues) > 0 {
		b.fields[fldName] = fieldTagValues
	}
}

func (b *structModelBuilder) populateByStruct(typeStruct *types.Struct) error {
	numFields := typeStruct.NumFields()
	for i := 0; i < numFields; i++ {
		fieldVar := typeStruct.Field(i)
		fldName := FieldName(fieldVar.Name())
		if fieldVar.IsField() {
			fieldType := fieldVar.Type()
			if fieldVar.Embedded() {
				if err := b.populateByType(fieldType); err != nil {
					return err
				}
			} else {
				tag := typeStruct.Tag(i)
				b.fieldNames = append(b.fieldNames, fldName)

				tagValues, fieldTagNames := parseTagValues(tag, b.includedTags)
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
						nestedBuilder := newBuilder(pkg, b.includedTags, b.deep)
						if err := nestedBuilder.populateByType(fieldTypeNamed); err != nil {
							return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
						}
						model := nestedBuilder.getModel(typeName, "")
						b.nested[fldName] = model
					}
				}

				b.fieldsType[fldName] = FieldType(fieldTypeStr)
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
	case *types.Named:
		underlying := tt.Underlying()
		return b.populateByType(underlying)
	default:
		return fmt.Errorf("unsupported type %s, type %v", tt, reflect.TypeOf(tt))
	}
}

func (b *structModelBuilder) newModel(typeName string, filePath string, t types.Type) (*StructModel, error) {
	if err := b.populateByType(t); err != nil {
		return nil, err
	}
	return b.getModel(typeName, filePath), nil
}

func (b *structModelBuilder) getModel(typeName string, filePath string) *StructModel {
	return &StructModel{
		TypeName:       typeName,
		FilePath:       filePath,
		PackageName:    b.pkg.Name(),
		PackagePath:    b.pkg.Path(),
		FieldsTagValue: b.fields,
		FieldNames:     b.fieldNames,
		FieldsType:     b.fieldsType,
		TagNames:       b.tagNames,
		Nested:         b.nested,
	}
}
