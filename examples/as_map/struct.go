package json

//go:generate fieldr -type Struct -output struct_as_map.go -wrap -export -AsMap -AsTagMap

type Struct struct {
	ID              int    `toMap:"id"`
	Name            string `toMap:"name"`
	Surname         string `toMap:"surname"`
	noExport        string `toMap:"no_export"` //nolint
	NoTag           string `toMap:""`
	IgnoredInTagMap string
}
