package examples

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func (s *Struct) MarshalJSON() ([]byte, error) {
	var builder strings.Builder

	fields := struct_TagValues[StructTag_json]
	values := s.AsTagMap(StructTag_json)

	builder.Grow(len(fields) * 16)

	err := writeJson(&builder, fields, values)
	if err != nil {
		return nil, err
	}

	return []byte(builder.String()), nil
}

func (s *Struct) MarshalJSONToBuilder(builder *strings.Builder) error {
	fields := struct_TagValues[StructTag_json]
	values := s.AsTagMap(StructTag_json)

	err := writeJson(builder, fields, values)
	if err != nil {
		return err
	}
	return nil
}

func writeJson(builder *strings.Builder, fields StructTagValues, values map[StructTagValue]interface{}) error {
	builder.WriteString("{")

	first := true
	for _, jsonField := range fields {
		fieldValue := values[jsonField]
		if !first {
			builder.WriteString(",")
		}

		builder.WriteString("\"")
		builder.WriteString(string(jsonField))
		builder.WriteString("\":")

		jsonValue, err := toJsonValue(fieldValue)
		if err != nil {
			return err
		}
		builder.WriteString(jsonValue)

		first = false
	}

	builder.WriteString("}")
	return nil
}

func toJsonValue(v interface{}) (string, error) {
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
		return "", errors.New(fmt.Sprintf("unsipported value %v, type %s", v, reflect.TypeOf(v)))
	}
}
