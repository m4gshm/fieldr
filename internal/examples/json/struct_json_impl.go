package json

import (
	json "encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	jsonFields     = structJsons()
	jsonFieldNames = make([]string, 0)
	jsonOmitEmpty  = make([]bool, 0)
)

func init() {
	for _, tag := range jsonFields {
		jsonFieldName := tag.field()
		omitEmpty := false

		strTag := string(tag)
		const omitEmptySuffix = ",omitempty"
		if strings.HasSuffix(strTag, omitEmptySuffix) {
			strTag = strTag[0 : len(strTag)-len(omitEmptySuffix)]
			omitEmpty = true
		}

		if strTag != "" {
			jsonFieldName = strTag
		}

		jsonFieldNames = append(jsonFieldNames, jsonFieldName)
		jsonOmitEmpty = append(jsonOmitEmpty, omitEmpty)
	}
}

func isEmpty(v interface{}) bool {
	if v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil()) {
		return true
	}
	switch vk := v.(type) {
	case string:
		return len(vk) == 0
	case bool:
		return !vk
	case int, int8, int16, int32, int64:
		return vk == 0
	case uint, uint8, uint16, uint32, uint64:
		return vk == 0
	case float32, float64:
		return vk == 0
	default:
		switch reflect.TypeOf(vk).Kind() {
		case reflect.Array, reflect.Map, reflect.Slice:
			return reflect.TypeOf(vk).Len() == 0
		}
		return v == nil
	}
}

func (s *Struct[S]) MarshalJSON() ([]byte, error) {
	var builder strings.Builder

	builder.Grow(len(jsonFields) * 16)

	err := s.MarshalJSONToBuilder(&builder)
	if err != nil {
		return nil, err
	}

	return []byte(builder.String()), nil
}

func (s *Struct[S]) MarshalJSONToBuilder(builder *strings.Builder) error {
	err := s.writeJson(builder)
	if err != nil {
		return err
	}
	return nil
}

func (s *Struct[S]) writeJson(builder *strings.Builder) error {
	builder.WriteString("{")
	j := 0
	for i, field := range jsonFields {
		fieldValue := s.val(field)

		if jsonOmitEmpty[i] && isEmpty(fieldValue) {
			continue
		}

		jsonValue, err := toJsonValue(fieldValue)
		if err != nil {
			return err
		}
		if len(jsonValue) == 0 {
			continue
		}

		if j > 0 {
			builder.WriteString(",")
		}

		builder.WriteString("\"")
		builder.WriteString(jsonFieldNames[i])
		builder.WriteString("\":")

		builder.WriteString(jsonValue)
		j++
	}

	builder.WriteString("}")
	return nil
}

func toJsonValue(v interface{}) (string, error) {
	if v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil()) {
		return "", nil
	}
	switch vt := v.(type) {
	case string:
		return "\"" + vt + "\"", nil
	case int:
		return strconv.FormatInt(int64(vt), 10), nil
	case int8:
		return strconv.FormatInt(int64(vt), 10), nil
	case int32:
		return strconv.FormatInt(int64(vt), 10), nil
	case int64:
		return strconv.FormatInt(vt, 10), nil
	case json.Marshaler:
		marshalJSON, err := vt.MarshalJSON()
		if err != nil {
			return "", err
		}
		return string(marshalJSON), nil

	default:

		return fmt.Sprintf("\"%v\"", v), nil
	}
}
