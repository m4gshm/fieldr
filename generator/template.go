package generator

type TemplateDataObject struct {
	Fields        []string
	Tags          []string
	FieldTags     map[string][]string
	TagValues     map[string][]string
	TagFields     map[string][]string
	FieldTagValue map[string]map[string]string
}
