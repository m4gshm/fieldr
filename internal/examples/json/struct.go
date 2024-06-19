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

type Struct[S string] struct {
	*MiddleStruct
	Name     NameS `json:"name,omitempty"`
	Surname  S     `json:"surname,omitempty"`
	NoJson   S     `json:"-"`
	noExport S     `json:"no_export"` //nolint
	NoTag    S
	Address  *Address `json:"address,omitempty"`
}

//go:generate fieldr -type Struct -out struct_util.go

//go:fieldr fields-to-consts -type structJson -val "rexp('[^-]+', OR(tag.json, field.name))" -list . -val-access . -field-name-access .
