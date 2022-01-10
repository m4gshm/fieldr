package json

type NameS string

type IDAware struct {
	ID int `json:"id"`
}

type BaseStruct struct {
	*IDAware
}

type MiddleStruct struct {
	BaseStruct
}

type Address struct {
	City, Street, Home string
}

type Struct struct {
	*MiddleStruct
	Name     NameS  `json:"name,omitempty"`
	Surname  string `json:"surname,omitempty"`
	NoJson   string `json:"-"`
	noExport string `json:"no_export"` //nolint
	NoTag    string
	Address  *Address `json:"address,omitempty"`
}

//go:generate fieldr -type Struct -out struct_util.go

//go:fieldr enum-const -type structJson -val "rexp \"[^-]+\" (OR tag.json field.name)" -list . -val-access
