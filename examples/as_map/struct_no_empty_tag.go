package json

//go:generate fieldr -type StructNoEmptyTag -output struct_no_empty_tag_as_map.go -wrap -export -AsMap -AsTagMap -noEmptyTag

type StructNoEmptyTag struct {
	ID      int    `toMap:"id"`
	Name    string `toMap:"name"`
	Surname string `toMap:"surname"`
	NoTag   string `toMap:""`
}
