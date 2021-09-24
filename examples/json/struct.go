package json

type Struct struct {
	ID       int    `json:"id"`
	Name     string `json:"name,omitempty"`
	Surname  string `json:"surname,omitempty"`
	NoJson   string `json:"-"`
	noExport string `json:"no_export"` //nolint
	NoTag    string
}

//go:generate fieldr -type Struct -export -output struct_util.go -Fields -FieldTagValueMap -GetFieldValue
