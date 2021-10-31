package as_map

//go:generate fieldr -type Struct -out struct_as_map.go -wrap -export -AsMap -AsTagMap

type BaseStruct struct {
	ID int `toMap:"id"`
}

type Struct struct {
	BaseStruct
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
}
