package struc

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"github.com/m4gshm/gollections/convert/as"
	"github.com/m4gshm/gollections/map_"
	// "github.com/m4gshm/gollections/op"
	"github.com/m4gshm/gollections/slice"

	"github.com/m4gshm/fieldr/logger"
	"github.com/m4gshm/fieldr/model/util"
)

type HandledStructs = map[types.Type]*Model

func NewModel(outPkgPath string, loopControl HandledStructs, typ util.TypeNamedOrAlias, typFile *ast.File) (*Model, error) {
	typName := typ.Obj().Name()
	if _, ok := loopControl[typ]; ok {
		return nil, fmt.Errorf("already handled type %v", typName)
	}
	model := &Model{
		Typ:            typ,
		TypFile:        typFile,
		OutPkgPath:     outPkgPath,
		FieldsTagValue: map[FieldName]map[TagName]TagValue{},
		TagsFieldValue: map[TagName]map[FieldName]TagValue{},
		FieldNames:     []FieldName{},
		FieldsType:     map[FieldName]FieldType{},
	}
	loopControl[typ] = model

	return model, (&structModelBuilder{
		deep:        true,
		outPkgPath:  outPkgPath,
		loopControl: loopControl,
	}).populateByStruct(model)
}

type structModelBuilder struct {
	deep        bool
	outPkgPath  string
	loopControl HandledStructs
}

func populateTags(model *Model, fieldName FieldName, tagName TagName, tagValue TagValue) {
	tagFields, tagFieldsOk := model.TagsFieldValue[tagName]
	if !tagFieldsOk {
		tagFields = make(map[FieldName]TagValue)
		model.TagsFieldValue[tagName] = tagFields
	}
	tagFields[fieldName] = tagValue
}

func populateFields(model *Model, fldName FieldName, fieldTagNames []TagName, tagValues map[TagName]TagValue) {
	fieldTagValues := slice.Map(fieldTagNames, as.Is[TagName], map_.Getter(tagValues))
	if len(fieldTagValues) > 0 {
		model.FieldsTagValue[fldName] = fieldTagValues
	}
}

func (b *structModelBuilder) populateByStruct(model *Model) error {
	deep := b.deep
	outPkgPath := b.outPkgPath
	loopControl := b.loopControl

	typFile := model.TypFile

	typeSpec := GetStructType(model.TypeName(), typFile)

	var fields []*ast.Field
	if typeSpec != nil {
		f := typeSpec.Fields
		if f != nil {
			fields = f.List
		}
	}

	typ := model.Typ
	obj := typ.Obj()

	strucTyp, _ := util.GetTypeStruct(typ)
	if strucTyp == nil {
		typName := obj.Name()
		return fmt.Errorf("'%s' is not a struct type", typName)
	}

	numFields := strucTyp.NumFields()
	for i := 0; i < numFields; i++ {
		field := slice.Get(fields, i)
		fieldVar := strucTyp.Field(i)
		if !fieldVar.IsField() {
			return fmt.Errorf("unexpected struct element, must be field, value %v, type %v", fieldVar, reflect.TypeOf(fieldVar))
		}
		// pos := fieldVar.Pos()
		// b.model.TypFile.Package

		fldName := fieldVar.Name()
		if _, ok := model.FieldsType[fldName]; ok {
			logger.Infof("duplicated field '%s'", fldName)
			continue
		}
		model.FieldNames = append(model.FieldNames, fldName)

		tagValues, fieldTagNames := parseTagValues(strucTyp.Tag(i))
		populateFields(model, fldName, fieldTagNames, tagValues)
		for _, fieldTagName := range fieldTagNames {
			populateTags(model, fldName, fieldTagName, tagValues[fieldTagName])
		}
		fieldType := fieldVar.Type()
		fieldTypeName := util.TypeString(fieldType, outPkgPath)
		importFieldTypeName := ""
		if field != nil {
			importFieldTypeName = types.ExprString(field.Type)
		}
		_ = importFieldTypeName

		// op.IfGetElse(field != nil,
		// 	func() string { return types.ExprString(field.Type) },
		// 	func() string { return util.TypeString(fieldType, outPkgPath) })

		refDeep := 0
		var fieldModel *Model
		if strucTyp, p := util.GetStructTypeNamed(fieldType); strucTyp != nil {
			strucTypObj := strucTyp.Obj()
			typeName := strucTypObj.Name()
			refDeep = p
			fieldTypeName = typeName
			if deep {
				if fmodel, ok := loopControl[strucTyp]; ok {
					logger.Debugf("found handled type %v", typeName)
					fieldModel = fmodel
				} else if model, err := NewModel(outPkgPath, loopControl, strucTyp, model.TypFile); err != nil {
					return fmt.Errorf("nested field %v.%v; %w", typeName, fldName, err)
				} else {
					fieldModel = model
				}
			}
		}
		model.FieldsType[fldName] = NewFieldType(fieldVar.Embedded(), refDeep, fieldTypeName, fieldType, fieldModel)
	}
	return nil
}

func GetStructType(typeName string, typFile *ast.File) *ast.StructType {
	var typeSpec *ast.StructType
	for _, d := range typFile.Decls {
		if genDecl, ok := d.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, s := range genDecl.Specs {
				if ts := s.(*ast.TypeSpec); ts.Name.Name == typeName {
					if st, ok := ts.Type.(*ast.StructType); ok {
						typeSpec = st
						break
					}
				}
			}
		}
	}
	return typeSpec
}

func NewFieldType(embedded bool, refDeep int, name string, fieldType types.Type, fieldModel *Model) FieldType {
	return FieldType{
		Embedded: embedded,
		RefDeep:  refDeep,
		Name:     name,
		Type:     fieldType,
		Model:    fieldModel,
	}
}
