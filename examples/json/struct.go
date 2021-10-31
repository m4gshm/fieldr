package json

type BaseStruct struct {
	ID int `json:"id"`
}

type Struct struct {
	BaseStruct
	Name     string `json:"name,omitempty"`
	Surname  string `json:"surname,omitempty"`
	NoJson   string `json:"-"`
	noExport string `json:"no_export"` //nolint
	NoTag    string
}

//go:generate fieldr -type Struct -export -out struct_util.go -Fields -FieldTagValueMap -GetFieldValue -EnumTags -EnumTagValues
