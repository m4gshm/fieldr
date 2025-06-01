package struc

import (
	"fmt"
	"go/types"

	"github.com/m4gshm/fieldr/model/util"
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
	FieldType struct {
		Embedded bool
		RefDeep  int
		Name     string
		Model    *Model
		Type     types.Type
	}

	//Model struct type model.
	Model struct {
		Typ            *types.Named
		typeName       string
		RefCount       int
		pkg            *types.Package
		OutPkgPath     string
		FieldsTagValue map[FieldName]map[TagName]TagValue
		TagsFieldValue map[TagName]map[FieldName]TagValue
		FieldNames     []FieldName
		FieldsType     map[FieldName]FieldType
	}
)

func (f *FieldType) FullName(outPkgPath string) string {
	return util.TypeString(f.Type, outPkgPath)
}

func (m *Model) FieldsNameAndType(yield func(FieldName, FieldType) bool) {
	for _, fn := range m.FieldNames {
		if !yield(fn, m.FieldsType[fn]) {
			break
		}
	}
}

func (m *Model) Package() *types.Package {
	return m.pkg
}

func (m *Model) TypeName() string {
	return m.typeName
}

// New - Model's default constructor.
func New(outPkgPath string, structType *types.Named) (*Model, error) {
	structModel, err := newBuilder(outPkgPath, handledStructs{}).newModel(structType)
	if err != nil {
		return nil, fmt.Errorf("new model of %+v: %w", structType, err)
	} else if structModel == nil {
		return nil, fmt.Errorf("nil model for type %+v", structType)
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
