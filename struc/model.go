package struc

import (
	"fmt"
	"go/types"
)

const ReplaceableValueSeparator = "="
const KeyValueSeparator = ":"
const ListValuesSeparator = ","

var (
	excludeValues = map[TagName]map[TagValue]bool{}
)

type (
	TagName   = string
	TagValue  = string
	FieldName = string
	Package   struct{ Name, Path string }
	FieldType struct {
		Embedded       bool
		RefCount       int
		Name, FullName string
		Model          *Model
		Type           types.Type
	}

	//Model struct type model.
	Model struct {
		Typ            *types.Named
		TypeName       string
		RefCount       int
		Package        Package
		OutPkgPath     string
		FieldsTagValue map[FieldName]map[TagName]TagValue
		TagsFieldValue map[TagName]map[FieldName]TagValue
		FieldNames     []FieldName
		FieldsType     map[FieldName]FieldType
	}
)

// New - Model's default constructor.
func New(outPkgPath string, structType *types.Named, typePkg Package) (*Model, error) {
	structModel, err := newBuilder(outPkgPath, handledStructs{}).newModel(typePkg, structType)
	if err != nil {
		return nil, fmt.Errorf("new model of %+v: %w", structType, err)
	} else if structModel == nil {
		return nil, fmt.Errorf("nil model for type %+v, package %+v", structType, typePkg)
	}
	return structModel, nil
}

func parseTagValues(tags string) (map[TagName]TagValue, []TagName) {
	tagNames := make([]TagName, 0)
	tagValues := make(map[TagName]TagValue)

	var prevTagPos int
	tagValueLen := len(tags)
	for pos := 0; pos < tagValueLen; pos++ {
		character := rune(tags[pos])
		switch character {
		case '`', ' ':
			prevTagPos = pos + 1
		case ':':
			_tagName := TagName(tags[prevTagPos:pos])

			//parse TagValue
			pos++

			character = rune(tags[pos])
			tagValueBorder := '"'
			findEndBorder := false
			if character == tagValueBorder {
				pos++
				findEndBorder = true
			}
			tagDelim := ' '

			var endValuePos int
			for endValuePos = pos; endValuePos < tagValueLen; endValuePos++ {
				character = rune(tags[endValuePos])
				if findEndBorder && character == tagValueBorder {
					break
				} else if character == tagDelim {
					break
				}
			}

			tagContent := tags[pos:endValuePos]
			var excluded bool
			if excludedValues, ok := excludeValues[_tagName]; ok {
				excluded, ok = excludedValues[tagContent]
				excluded = excluded && ok
			}

			if !excluded {
				tagValues[_tagName] = tagContent
				tagNames = append(tagNames, _tagName)
			}

			prevTagPos = endValuePos
			pos = prevTagPos
		}
	}
	return tagValues, tagNames
}
